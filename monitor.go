package gta

import (
	"time"

	"github.com/jinzhu/gorm"
)

type taskMonitor interface {
	GoMonitorBuiltinTasks()
}

type taskMonitorImp struct {
	config    *Config
	register  taskRegister
	dal       taskDAL
	assembler taskAssembler
}

func (s *taskMonitorImp) GoMonitorBuiltinTasks() {
	logger := s.config.logger()
	for _, key := range s.register.GetBuiltInKeys() {
		taskDef, err := s.register.GetDefinition(key)
		if err != nil {
			logger.Errorf("[GoMonitorBuiltinTasks] get task definition failed, err[%v], task_key[%v]", err, key)
		}
		s.goMonitorBuiltinTask(taskDef)
	}
}

func (s *taskMonitorImp) goMonitorBuiltinTask(taskDef *TaskDefinition) {
	logger := s.config.logger()
	logger.Infof("monitor built-in task start, task_key[%v], monitor interval[%v]", taskDef.key, taskDef.loopInterval)
	go func() {
		defer panicHandler()
		for {
			select {
			case <-s.config.done():
				return
			default:
				s.monitorBuiltinTask(taskDef)
				time.Sleep(randomInterval(taskDef.loopInterval))
			}
		}
	}()
}

func (s *taskMonitorImp) monitorBuiltinTask(taskDef *TaskDefinition) {
	logger := s.config.logger()

	newTask, err := s.assembler.AssembleTask(s.config.Context, taskDef, taskDef.argument)
	if err != nil {
		logger.Errorf("[monitorBuiltinTask] assemble buitin task failed, err[%v], task_key[%v]", err, taskDef.key)
		return
	}
	newTask.TaskStatus = taskStatusInitialized

	// try to update
	updateFunc := func(tx *gorm.DB) error {
		if task, err := s.dal.GetForUpdate(tx, taskDef.taskID); err != nil { // other error
			return err
		} else if s.needLoopBuiltinTask(task, taskDef) {
			if rows, err := s.dal.UpdateByIDAndKey(tx, newTask.ID, newTask.TaskKey, newTask.updateMap()); err != nil {
				return err
			} else if rows <= 0 {
				return ErrNotUpdated
			}
		}
		return nil
	}

	if err := s.config.DBFactory().Transaction(updateFunc); err == gorm.ErrRecordNotFound { // need create //TODO
		_ = s.dal.CreateTask(s.config.DBFactory(), newTask) // ignore primary key conflict
	} else if err != nil {
		logger.Errorf("[monitorBuiltinTask] update transaction failed, err[%v], task_key[%v]", err, taskDef.key)
		return
	}
}

func (s *taskMonitorImp) needLoopBuiltinTask(task TaskModel, taskDef *TaskDefinition) bool {
	// normal loop if task_status is succeeded or failed
	needNormalLoop := time.Since(task.UpdatedAt) >= taskDef.loopInterval && (task.
		TaskStatus == taskStatusSucceeded || task.TaskStatus == taskStatusFailed)

	// force loop if abnormal running found
	needForceLoop := time.Since(task.UpdatedAt) >= s.config.RunningTimeout && task.TaskStatus == taskStatusRunning

	return needNormalLoop || needForceLoop
}
