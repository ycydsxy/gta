package gta

import (
	"sync/atomic"
	"time"
)

type taskScanner interface {
	GoScanAndSchedule()
}

type taskScannerImp struct {
	config      *TaskConfig
	register    taskRegister
	dal         taskDAL
	scheduler   taskScheduler
	instantScan atomic.Value
}

func (s *taskScannerImp) GoScanAndSchedule() {
	logger := s.config.logger()
	logger.Infof("[GoScanAndSchedule] scan and run start, scan interval[%v], instant scan interval[%v]",
		s.config.ScanInterval, s.config.InstantScanInvertal)
	go func() {
		defer panicHandler()
		for {
			select {
			case <-s.config.done():
				return
			default:
				s.scanAndSchedule()
				time.Sleep(s.scanInterval())
			}
		}
	}()
}

func (s *taskScannerImp) scanAndSchedule() {
	logger := s.config.logger()

	if !s.scheduler.CanSchedule() {
		// the schedule has reached its capacity limit
		s.switchOffInstantScan()
		return
	}

	task, err := s.claimInitializedTask()
	if err != nil {
		// no task remained or other error occurred, i.e. the db has gone
		if err != ErrTaskNotFound {
			logger.Errorf("[scanAndSchedule] claim task err, err[%v]", err)
		}
		s.switchOffInstantScan()
		return
	} else if task != nil {
		s.scheduler.GoScheduleTask(task)
	}

	s.switchOnInstantScan()
}

func (s *taskScannerImp) scanInterval() time.Duration {
	if s.needInstantScan() {
		return randomInterval(s.config.InstantScanInvertal)
	}
	return randomInterval(s.config.ScanInterval)
}

func (s *taskScannerImp) needInstantScan() bool {
	iv := s.instantScan.Load()
	if iv == nil {
		return false
	}
	return iv.(bool)
}

func (s *taskScannerImp) switchOffInstantScan() {
	s.instantScan.Store(false)
}

func (s *taskScannerImp) switchOnInstantScan() {
	s.instantScan.Store(true)
}

func (s *taskScannerImp) claimInitializedTask() (*Task, error) {
	c := s.config

	sensitiveKeys, insensitiveKeys := s.register.GroupKeysByInitTimeoutSensitivity()
	task, err := s.dal.GetInitialized(c.DB, sensitiveKeys, c.InitializedTimeout, insensitiveKeys)
	if err != nil {
		return nil, err
	} else if task == nil {
		// no initialized tasks remained
		return nil, ErrTaskNotFound
	}

	select {
	case <-c.done():
		// abort claim when cancel signal received
		return nil, nil
	default:
		if rowsAffected, err := s.dal.UpdateStatusByIDs(c.DB, []uint64{task.ID}, task.TaskStatus, TaskStatusRunning); err != nil {
			return nil, err
		} else if rowsAffected == 0 {
			// task is claimed by others, ignore error
			return nil, nil
		}
		task.TaskStatus = TaskStatusRunning
		return task, nil
	}
}
