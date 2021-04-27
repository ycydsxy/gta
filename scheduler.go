package gta

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/panjf2000/ants/v2"
)

const (
	transactionKey = "task/async:transaction"
)

type taskScheduler interface {
	Transaction(fc func(tx *gorm.DB) error) error
	CreateTask(tx *gorm.DB, ctxIn context.Context, key TaskKey, arg interface{}) error
	Stop(wait bool)
	GoScheduleTask(task *TaskModel)
	CanSchedule() bool
	StartRunning(task *TaskModel) error
}

type taskSchedulerImp struct {
	config     *Config
	register   taskRegister
	dal        taskDAL
	assembler  taskAssembler
	pool       *ants.Pool
	runningMap sync.Map
}

func (s *taskSchedulerImp) Transaction(fc func(tx *gorm.DB) error) error {
	db := s.config.DBFactory().InstantSet(transactionKey, &sync.Map{})

	if err := db.Transaction(fc); err != nil {
		return err
	}

	toScheduleTasks, ok := db.Get(transactionKey)
	if !ok {
		return ErrUnexpected
	}

	toScheduleTasks.(*sync.Map).Range(func(key, value interface{}) bool {
		s.GoScheduleTask(value.(*TaskModel))
		return true
	})

	return nil
}

func (s *taskSchedulerImp) CreateTask(tx *gorm.DB, ctxIn context.Context, key TaskKey, arg interface{}) error {
	logger := s.config.LoggerFactory(ctxIn)

	taskDef, err := s.register.GetDefinition(key)
	if err != nil {
		return err // TODO
	}

	task, err := s.assembler.AssembleTask(ctxIn, taskDef, arg)
	if err != nil {
		return err // TODO
	}

	select {
	case <-s.config.done(): // may still accept run task request when waiting in the stop process
		if err := s.createInitializedTask(tx, task); err != nil {
			return err // TODO
		}
	default:
		if toScheduleTasks, ok := tx.Get(transactionKey); ok { // buitin transaction, create running task
			if !s.config.DryRun {
				if s.CanSchedule() {
					if err := s.createRunningTask(tx, task); err != nil {
						return err // TODO
					}
					if _, loaded := toScheduleTasks.(*sync.Map).LoadOrStore(task.ID, task); loaded {
						return ErrUnexpected
					}
				} else {
					if err := s.createInitializedTask(tx, task); err != nil {
						return err // TODO
					}
				}
			} else {
				// task will be scheduled after the transaction succeeded
				if _, loaded := toScheduleTasks.(*sync.Map).LoadOrStore(task.ID, task); loaded {
					return ErrUnexpected
				}
			}
		} else { // not builtin transaction, create initialized task
			if !s.config.DryRun {
				if err := s.createInitializedTask(tx, task); err != nil {
					return err // TODO
				}
			} else {
				logger.Warnf("[CreateTask] Using dry run mode in non-builtin transaction, this task will be scheduled before the transaction is commited!")
				go func() {
					// wait for commiting the transaction in dry run mode
					time.Sleep(time.Millisecond * 500)
					s.GoScheduleTask(task)
				}()
			}
		}
	}
	logger.Infof("[CreateTask] async task created, task_key[%v], task_id[%v], task_status[%v]", key, task.ID, task.TaskStatus)
	return nil
}

func (s *taskSchedulerImp) Stop(wait bool) {
	defer s.pool.Release()

	logger := s.config.logger()

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
		} else if !wait || (s.config.WaitTimeout > 0 && time.Since(waitStart) > s.config.WaitTimeout) {
			// change remaining tasks status to initialized
			rowsAffected, err := s.dal.UpdateTaskStatusByIDs(s.config.DBFactory(), taskIDs, taskStatusRunning, taskStatusInitialized)
			if err != nil {
				logger.Errorf("[Stop] update task status from running to initialized failed, err[%v]", err)
				return
			}
			logger.Infof("[Stop] change current running tasks to initialized, len[%v], changed len[%v]", len(taskIDs), rowsAffected)
			return
		}
	}
}

func (s *taskSchedulerImp) GoScheduleTask(task *TaskModel) {
	logger := s.config.logger()

	f := func() {
		defer panicHandler()
		s.scheduleTask(task)
	}

	if err := s.pool.Submit(f); err == ants.ErrPoolOverload {
		// We really don't want to be blocked in this function. However, a bloking or an error is inevitable when the
		// pool is full. Under these circumstances, we choose to create tasks in 'initialized' status and suspend the
		// scan process so that we won't enter this branch in most cases.
		logger.Warnf("[GoScheduleTask] schedule by extra goroutine, task_key[%v], task_id[%v]", task.TaskKey, task.ID)
		go f()
	} else if err != nil {
		logger.Errorf("[GoScheduleTask] schedule task failed, err[%v], task_key[%v], task_id[%v]", err, task.TaskKey, task.ID)
		return
	}
}

func (s *taskSchedulerImp) CanSchedule() bool {
	return s.pool.Free() > 0
}

func (s *taskSchedulerImp) scheduleTask(task *TaskModel) {
	logger := s.config.logger()

	taskDef, err := s.register.GetDefinition(task.TaskKey)
	if err != nil { // TODO
		logger.Errorf("[scheduleTask] get task definition failed, err[%v], task_key[%v], task_id[%v]", err, task.TaskKey, task.ID)
		return
	}

	succeeded := false
	defer func() {
		var toStatus TaskStatus
		if succeeded {
			toStatus = taskStatusSucceeded
			logger.Infof("[scheduleTask] schedule task succeeded, task_key[%v], task_id[%v]", task.TaskKey, task.ID)
		} else {
			toStatus = taskStatusFailed
			logger.Errorf("[scheduleTask] schedule task failed, task_key[%v], task_id[%v]", task.TaskKey, task.ID)
		}
		if s.config.DryRun {
			return
		}
		if err := s.stopRunning(task, taskDef, toStatus); err != nil {
			logger.Errorf("[scheduleTask] change running task status error, err[%v], task_key[%v], task_id[%v]", err, task.TaskKey, task.ID)
		}
	}()

	for cur := 0; cur <= taskDef.RetryTimes; cur++ {
		if cur != 0 {
			time.Sleep(s.retryInterval(cur))
			logger.Warnf("[scheduleTask] start retry, current retry times[%v], task_key[%v], task_id[%v]", cur, task.TaskKey, task.ID)
		}
		cost, err := s.executeTask(taskDef, task)
		if err == nil {
			logger.Infof("[scheduleTask] execute task handler succeeded, cost[%v], task_key[%v], task_id[%v]", cost, task.TaskKey, task.ID)
			succeeded = true
			break
		} else {
			logger.Warnf("[scheduleTask] execute task handler failed, cost[%v], err[%v], task_key[%v], task_id[%v]", cost, err, task.TaskKey, task.ID)
		}
	}
}

func (s *taskSchedulerImp) executeTask(taskDef *TaskDefinition, task *TaskModel) (cost time.Duration, err error) {
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
		}
		cost = time.Since(startTime).Round(time.Millisecond)
	}()

	ctxIn, argument, tempErr := s.assembler.DisassembleTask(taskDef, task)
	if tempErr != nil {
		err = fmt.Errorf("disassemble task error: %w", tempErr)
		return
	}

	if tempErr := taskDef.Handler(ctxIn, argument); tempErr != nil {
		err = fmt.Errorf("handle failed: %w", tempErr)
		return
	}

	return
}

func (s *taskSchedulerImp) stopRunning(task *TaskModel, taskDef *TaskDefinition, toStatus TaskStatus) error {
	if task.TaskStatus != taskStatusRunning {
		return fmt.Errorf("task status[%v] is not running", task.TaskStatus)
	}
	if taskDef.CleanSucceeded && toStatus == taskStatusSucceeded {
		if rowsAffected, err := s.dal.HardDeleteTaskByIDAndStatus(s.config.DBFactory(), task.ID,
			task.TaskStatus); err != nil {
			return fmt.Errorf("clean task error: %v", err)
		} else if rowsAffected == 0 {
			return ErrNotUpdated
		}
	} else {
		if rowsAffected, err := s.dal.UpdateTaskStatus(s.config.DBFactory(), *task, toStatus); err != nil {
			return fmt.Errorf("update task status from %v to %v error: %v", task.TaskStatus, toStatus, err)
		} else if rowsAffected == 0 {
			return ErrNotUpdated
		}
	}
	task.TaskStatus = toStatus
	s.unmarkRunning(task)
	return nil
}

func (s *taskSchedulerImp) StartRunning(task *TaskModel) error {
	if task.TaskStatus == taskStatusRunning {
		return fmt.Errorf("task status is already running")
	}
	if rowsAffected, err := s.dal.UpdateTaskStatus(s.config.DBFactory(), *task, taskStatusRunning); err != nil {
		return fmt.Errorf("update task status from %v to %v error: %v", task.TaskStatus, taskStatusRunning, err)
	} else if rowsAffected == 0 {
		return ErrNotUpdated
	}
	task.TaskStatus = taskStatusRunning
	s.markRunning(task)
	return nil
}

func (s *taskSchedulerImp) createInitializedTask(tx *gorm.DB, task *TaskModel) error {
	task.TaskStatus = taskStatusInitialized
	return s.dal.CreateTask(tx, task)
}

func (s *taskSchedulerImp) createRunningTask(tx *gorm.DB, task *TaskModel) error {
	task.TaskStatus = taskStatusRunning
	if err := s.dal.CreateTask(tx, task); err != nil {
		return err
	}
	s.markRunning(task)
	return nil
}

func (s *taskSchedulerImp) markRunning(task *TaskModel) {
	s.runningMap.Store(task.ID, nil)
}

func (s *taskSchedulerImp) unmarkRunning(task *TaskModel) {
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

func (s *taskSchedulerImp) retryInterval(currentTimes int) time.Duration {
	switch currentTimes {
	case 1:
		return time.Second * 1
	case 2:
		return time.Second * 5
	case 3:
		return time.Second * 30
	}
	return time.Minute
}
