package gta

import (
	"sync/atomic"
	"time"

	"github.com/jinzhu/gorm"
)

type taskScanner interface {
	GoScanAndSchedule()
}

type taskScannerImp struct {
	config    *Config
	register  taskRegister
	dal       taskDAL
	scheduler taskScheduler

	instantScan atomic.Value
}

func (s *taskScannerImp) GoScanAndSchedule() {
	logger := s.config.logger()
	logger.Infof("scan and run start, scan interval[%v]", s.config.ScanInterval)
	go func() {
		defer panicHandler()
		for {
			select {
			case <-s.config.done():
				return
			default:
				s.scanAndSchedule()
				time.Sleep(s.getScanInterval())
			}
		}
	}()
}

func (s *taskScannerImp) scanAndSchedule() {
	logger := s.config.logger()

	if !s.scheduler.CanSchedule() {
		s.switchOffInstantScan()
		return
	}

	task, err := s.claimInitializedTask()
	if err == ErrTaskNotFound {
		// no task remained
		s.switchOffInstantScan()
		return
	} else if err != nil {
		logger.Errorf("[scanAndSchedule] claim task err, err[%v]", err)
		// maybe the db has gone, it's better to switch off instant scan
		s.switchOffInstantScan()
		return
	}

	s.switchOnInstantScan()
	if task != nil {
		s.scheduler.GoScheduleTask(task)
	}
}

func (s *taskScannerImp) getScanInterval() time.Duration {
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

func (s *taskScannerImp) claimInitializedTask() (*TaskModel, error) {
	sensitiveKeys, insensitiveKeys := s.register.GroupKeysByInitTimeoutSensitivity()
	task, err := s.dal.GetInitializedTask(s.config.SlaveDBFactory(), sensitiveKeys, s.config.InitializedTimeout,
		insensitiveKeys)
	if err == gorm.ErrRecordNotFound { // no initialized tasks remained
		return nil, ErrTaskNotFound
	} else if err != nil {
		return nil, err
	}

	select {
	case <-s.config.done(): // abort claim in stop process
		return nil, nil
	default:
		if err := s.scheduler.StartRunning(&task); err == ErrNotUpdated {
			// task is claimed by other pod, ignore error
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		return &task, nil
	}
}
