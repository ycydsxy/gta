package gta

import (
	"time"

	"gorm.io/gorm"
)

type taskMonitor interface {
	GoMonitorBuiltinTasks()
}

type taskMonitorImp struct {
	*options
	register  taskRegister
	dal       taskDAL
	assembler taskAssembler
}

func (s *taskMonitorImp) GoMonitorBuiltinTasks() {
	logger := s.logger()
	for _, key := range s.register.GetBuiltInKeys() {
		taskDef, _ := s.register.GetDefinition(key)
		s.goMonitorBuiltinTask(taskDef)
		logger.Infof("[GoMonitorBuiltinTasks] monitor built-in task start, task_key[%v], monitor interval[%v]", taskDef.key, taskDef.loopInterval)
	}
}

func (s *taskMonitorImp) goMonitorBuiltinTask(taskDef *TaskDefinition) {
	go func() {
		defer panicHandler()
		for {
			select {
			case <-s.done():
				return
			default:
				s.monitorBuiltinTask(taskDef)
				time.Sleep(randomInterval(taskDef.loopInterval))
			}
		}
	}()
}

func (s *taskMonitorImp) monitorBuiltinTask(taskDef *TaskDefinition) {
	logger := s.logger()

	newTask, err := s.assembler.AssembleTask(s.context, taskDef, taskDef.argument)
	if err != nil {
		logger.Errorf("[monitorBuiltinTask] assemble buitin task failed, err[%v], task_key[%v]", err, taskDef.key)
		return
	}
	newTask.TaskStatus = TaskStatusInitialized

	if err := s.getDB().Transaction(func(tx *gorm.DB) error {
		if task, err := s.dal.GetForUpdate(tx, taskDef.taskID); err != nil {
			return err
		} else if task == nil {
			return ErrTaskNotFound
		} else if !s.needLoopBuiltinTask(task, taskDef) {
			return nil
		}

		// need loop
		if rows, err := s.dal.Update(tx, newTask); err != nil {
			return err
		} else if rows <= 0 {
			return ErrZeroRowsAffected
		}
		return nil
	}); err == ErrTaskNotFound {
		// need create, ignore primary key conflict
		// TODO: distinguish primary key conflict error
		_ = s.dal.Create(s.getDB(), newTask)
		return
	} else if err != nil {
		logger.Errorf("[monitorBuiltinTask] update transaction failed, err[%v], task_key[%v]", err, taskDef.key)
		return
	}
}

func (s *taskMonitorImp) needLoopBuiltinTask(task *Task, taskDef *TaskDefinition) bool {
	// normal loop if task_status is succeeded or failed
	needNormalLoop := time.Since(task.UpdatedAt) >= taskDef.loopInterval && (task.TaskStatus == TaskStatusSucceeded || task.TaskStatus == TaskStatusFailed)
	// force loop if abnormal running found
	needForceLoop := time.Since(task.UpdatedAt) >= s.runningTimeout && task.TaskStatus == TaskStatusRunning

	return needNormalLoop || needForceLoop
}
