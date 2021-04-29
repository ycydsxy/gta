package gta

import (
	"time"
)

type taskScanner interface {
	GoScanAndSchedule()
}

type taskScannerImp struct {
	config      *Config
	register    taskRegister
	dal         taskDAL
	scheduler   taskScheduler
	instantScan bool
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
		// no task remained or other error occured, i.e. the db has gone
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
	if s.instantScan {
		return randomInterval(s.config.InstantScanInvertal)
	}
	return randomInterval(s.config.ScanInterval)
}

func (s *taskScannerImp) switchOffInstantScan() {
	s.instantScan = false
}

func (s *taskScannerImp) switchOnInstantScan() {
	s.instantScan = true
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
		if err := s.scheduler.StartRunning(task); err == ErrNotUpdated {
			// task is claimed by others, ignore error
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		return task, nil
	}
}
