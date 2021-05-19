package gta

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"gorm.io/gorm"
)

const (
	transactionKey = "gta:transaction"
)

type taskScheduler interface {
	Transaction(fc func(tx *gorm.DB) error) error
	CreateTask(tx *gorm.DB, ctxIn context.Context, key TaskKey, arg interface{}) error
	Stop(wait bool)
	GoScheduleTask(task *Task)
	CanSchedule() bool
}

type taskSchedulerImp struct {
	*options
	register   taskRegister
	dal        taskDAL
	assembler  taskAssembler
	pool       *ants.Pool
	runningMap sync.Map
}

func (s *taskSchedulerImp) Transaction(fc func(tx *gorm.DB) error) error {
	db := s.getDB().Set(transactionKey, &sync.Map{})

	if err := db.Transaction(fc); err != nil {
		return err
	}

	toScheduleTasks, _ := db.Get(transactionKey)
	toScheduleTasks.(*sync.Map).Range(func(key, value interface{}) bool {
		s.GoScheduleTask(value.(*Task))
		return true
	})

	return nil
}

func (s *taskSchedulerImp) CreateTask(tx *gorm.DB, ctxIn context.Context, key TaskKey, arg interface{}) error {
	logger := s.loggerFactory(ctxIn)

	taskDef, err := s.register.GetDefinition(key)
	if err != nil {
		return err
	}
	task, err := s.assembler.AssembleTask(ctxIn, taskDef, arg)
	if err != nil {
		return err
	}

	select {
	case <-s.done():
		// may still accept create task requests when cancel signal is received
		if err := s.createInitializedTask(tx, task); err != nil {
			return err
		}
	default:
		if toScheduleTasks, ok := tx.Get(transactionKey); ok {
			// buitin transaction, try to create running task
			if !s.dryRun {
				if s.CanSchedule() {
					if err := s.createRunningTask(tx, task); err != nil {
						return err
					}
					toScheduleTasks.(*sync.Map).Store(task.ID, task)
				} else {
					if err := s.createInitializedTask(tx, task); err != nil {
						return err
					}
				}
			} else {
				// assign a dummy id to this task in dry run mode
				task.ID = rand.Uint64()
				task.TaskStatus = TaskStatusRunning
				// task will be scheduled after the transaction succeeded
				toScheduleTasks.(*sync.Map).Store(task.ID, task)
			}
		} else {
			// not builtin transaction, create initialized task
			if !s.dryRun {
				if err := s.createInitializedTask(tx, task); err != nil {
					return err
				}
			} else {
				logger.Warnf("[CreateTask] Using dry run mode in non-builtin transaction, this task may be scheduled before the transaction is committed!")
				// assign a dummy id to this task in dry run mode
				task.ID = rand.Uint64()
				task.TaskStatus = TaskStatusRunning
				go func() {
					// wait for committing the transaction in dry run mode
					time.Sleep(time.Millisecond * 500)
					// note that the task will still be scheduled when the transaction is rolled back in dry run mode
					s.GoScheduleTask(task)
				}()
			}
		}
	}
	logger.Infof("[CreateTask] async task created in transaction, task_key[%v], task_id[%v], task_status[%v]", key, task.ID, task.TaskStatus)
	return nil
}

func (s *taskSchedulerImp) Stop(wait bool) {
	defer s.pool.Release()
	logger := s.logger()

	// first check, if tasks len is zero, return immediately
	taskIDs := s.runningTaskIDs()
	if len(taskIDs) <= 0 {
		return
	}
	// loop check and wait
	waitStart := time.Now()
	for {
		logger.Infof("[Stop] current running tasks len[%v], waiting...", len(taskIDs))
		time.Sleep(5 * time.Second)

		taskIDs = s.runningTaskIDs()
		if len(taskIDs) <= 0 {
			logger.Infof("[Stop] current running tasks finished")
			return
		} else if !wait || (s.waitTimeout > 0 && time.Since(waitStart) > s.waitTimeout) {
			if !s.dryRun {
				// change remaining tasks status to initialized
				rowsAffected, err := s.dal.UpdateStatusByIDs(s.getDB(), taskIDs, TaskStatusRunning, TaskStatusInitialized)
				if err != nil {
					logger.Errorf("[Stop] update task status from running to initialized failed, err[%v]", err)
					return
				}
				logger.Infof("[Stop] change current running tasks to initialized, len[%v], changed len[%v]", len(taskIDs), rowsAffected)
			} else {
				logger.Warnf("[Stop] remaining running tasks len[%v] in dry run mode, cannot gracefully exit!", len(taskIDs))
			}
			return
		}
	}
}

func (s *taskSchedulerImp) GoScheduleTask(task *Task) {
	logger := s.logger()

	if task.TaskStatus != TaskStatusRunning {
		logger.Errorf("[GoScheduleTask] invalid task status, task_key[%v], task_status[%v]", task.TaskKey, task.TaskStatus)
		return
	}
	s.markRunning(task)

	f := func() {
		defer panicHandler()
		s.scheduleTask(task)
	}

	if err := s.pool.Submit(f); err == ants.ErrPoolOverload {
		// We really don't want to be blocked in this function. However, a bloking or an error is inevitable when the
		// pool is full. Under these circumstances, we choose to create tasks in 'initialized' status and suspend the
		// scan process so that we won't enter this branch in most cases.
		go f()
	} else if err != nil {
		s.unmarkRunning(task)
		logger.Errorf("[GoScheduleTask] schedule task failed, err[%v], task_key[%v], task_id[%v]", err, task.TaskKey, task.ID)
		return
	}
}

func (s *taskSchedulerImp) CanSchedule() bool {
	return s.pool.Free() > 0
}

func (s *taskSchedulerImp) scheduleTask(task *Task) {
	logger := s.logger()
	taskDef, _ := s.register.GetDefinition(task.TaskKey)
	succeeded := false
	startTime := time.Now()
	logger.Infof("[scheduleTask] schedule task start, task_key[%v], task_id[%v]", task.TaskKey, task.ID)

	defer func() {
		var toStatus TaskStatus
		cost := time.Since(startTime).Round(time.Millisecond)
		if succeeded {
			toStatus = TaskStatusSucceeded
			logger.Infof("[scheduleTask] schedule task succeeded, cost[%v], task_key[%v], task_id[%v]", cost, task.TaskKey, task.ID)
		} else {
			toStatus = TaskStatusFailed
			logger.Errorf("[scheduleTask] schedule task failed, cost[%v], task_key[%v], task_id[%v]", cost, task.TaskKey, task.ID)
		}
		if err := s.stopRunning(task, taskDef, toStatus); err != nil {
			logger.Errorf("[scheduleTask] change running task status error, err[%v], task_key[%v], task_id[%v]", err, task.TaskKey, task.ID)
		}
		task.TaskStatus = toStatus
		s.unmarkRunning(task)
	}()

	for times := 0; times <= taskDef.RetryTimes; times++ {
		if times > 0 {
			time.Sleep(taskDef.retryInterval(times))
			logger.Warnf("[scheduleTask] start retry, current retry times[%v], task_key[%v], task_id[%v]", times, task.TaskKey, task.ID)
		}
		if err := s.executeTask(taskDef, task); err == nil {
			succeeded = true
			break
		}
	}
}

func (s *taskSchedulerImp) executeTask(taskDef *TaskDefinition, task *Task) (err error) {
	logger := s.logger()

	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
		}
		cost := time.Since(startTime).Round(time.Millisecond)
		if err == nil {
			logger.Infof("[executeTask] task handler succeeded, cost[%v], task_key[%v], task_id[%v]", cost, task.TaskKey, task.ID)
		} else {
			logger.Warnf("[executeTask] task handler failed, cost[%v], err[%v], task_key[%v], task_id[%v]", cost, err, task.TaskKey, task.ID)
		}
	}()

	logger.Infof("[executeTask] task handler start, task_key[%v], task_id[%v]", task.TaskKey, task.ID)
	ctxIn, argument, tempErr := s.assembler.DisassembleTask(taskDef, task)
	if tempErr != nil {
		err = fmt.Errorf("disassemble task error: %w", tempErr)
		return
	}
	if tempErr := taskDef.Handler(ctxIn, argument); tempErr != nil {
		err = fmt.Errorf("handle failed: %w", tempErr)
		return
	}

	return nil
}

func (s *taskSchedulerImp) stopRunning(task *Task, taskDef *TaskDefinition, toStatus TaskStatus) error {
	if !s.dryRun {
		if taskDef.CleanSucceeded && toStatus == TaskStatusSucceeded {
			if rowsAffected, err := s.dal.DeleteByIDAndStatus(s.getDB(), task.ID, task.TaskStatus); err != nil {
				return err
			} else if rowsAffected == 0 {
				return ErrZeroRowsAffected
			}
		} else {
			if rowsAffected, err := s.dal.UpdateStatusByIDs(s.getDB(), []uint64{task.ID}, task.TaskStatus, toStatus); err != nil {
				return err
			} else if rowsAffected == 0 {
				return ErrZeroRowsAffected
			}
		}
	}
	return nil
}

func (s *taskSchedulerImp) createInitializedTask(tx *gorm.DB, task *Task) error {
	task.TaskStatus = TaskStatusInitialized
	return s.dal.Create(tx, task)
}

func (s *taskSchedulerImp) createRunningTask(tx *gorm.DB, task *Task) error {
	task.TaskStatus = TaskStatusRunning
	return s.dal.Create(tx, task)
}

func (s *taskSchedulerImp) markRunning(task *Task) {
	s.runningMap.Store(task.ID, nil)
}

func (s *taskSchedulerImp) unmarkRunning(task *Task) {
	s.runningMap.Delete(task.ID)
}

func (s *taskSchedulerImp) runningTaskIDs() []uint64 {
	var res []uint64
	s.runningMap.Range(func(key, value interface{}) bool {
		res = append(res, key.(uint64))
		return true
	})
	return res
}
