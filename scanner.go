package gta

import (
	"sync/atomic"
	"time"
)

type taskScanner interface {
	GoScanAndSchedule()
}

type taskScannerImp struct {
	*options
	register    taskRegister
	dal         taskDAL
	scheduler   taskScheduler
	instantScan atomic.Value
}

func (s *taskScannerImp) GoScanAndSchedule() {
	logger := s.logger()
	logger.Infof("[GoScanAndSchedule] scan and run start, scan interval[%v], instant scan interval[%v]", s.scanInterval, s.instantScanInterval)
	go func() {
		defer panicHandler()
		for {
			select {
			case <-s.done():
				return
			default:
				s.scanAndSchedule()
				time.Sleep(s.randomScanInterval())
			}
		}
	}()
}

func (s *taskScannerImp) scanAndSchedule() {
	logger := s.logger()

	if !s.scheduler.CanSchedule() {
		// the schedule has reached its capacity limit
		s.swishOffInstantScan()
		return
	}

	task, err := s.claimInitializedTask()
	if err != nil {
		// no task remained or other error occurred, i.e. the db has gone
		if err != ErrTaskNotFound {
			logger.Errorf("[scanAndSchedule] claim task err, err[%v]", err)
		}
		s.swishOffInstantScan()
		return
	} else if task != nil {
		s.scheduler.GoScheduleTask(task)
	}

	s.swishOnInstantScan()
}

func (s *taskScannerImp) randomScanInterval() time.Duration {
	if s.needInstantScan() {
		return randomInterval(s.instantScanInterval)
	}
	return randomInterval(s.scanInterval)
}

func (s *taskScannerImp) needInstantScan() bool {
	iv := s.instantScan.Load()
	if iv == nil {
		return false
	}
	return iv.(bool)
}

func (s *taskScannerImp) swishOffInstantScan() {
	s.instantScan.Store(false)
}

func (s *taskScannerImp) swishOnInstantScan() {
	s.instantScan.Store(true)
}

func (s *taskScannerImp) claimInitializedTask() (*Task, error) {
	sensitiveKeys, insensitiveKeys := s.register.GroupKeysByInitTimeoutSensitivity()
	task, err := s.dal.GetInitialized(s.getDB(), sensitiveKeys, s.initializedTimeout, insensitiveKeys)
	if err != nil {
		return nil, err
	} else if task == nil {
		// no initialized tasks remained
		return nil, ErrTaskNotFound
	}

	select {
	case <-s.done():
		// abort claim when cancel signal received
		return nil, nil
	default:
		if rowsAffected, err := s.dal.UpdateStatusByIDs(s.getDB(), []uint64{task.ID}, task.TaskStatus, TaskStatusRunning); err != nil {
			return nil, err
		} else if rowsAffected == 0 {
			// task is claimed by others, ignore error
			return nil, nil
		}
		task.TaskStatus = TaskStatusRunning
		return task, nil
	}
}
