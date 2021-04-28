package gta

import (
	"context"
	"reflect"
	"time"
)

type TaskHandler func(ctx context.Context, arg interface{}) (err error)

type TaskDefinition struct {
	// must provide, task handler
	Handler TaskHandler

	// optional, task argument type in the handler
	ArgType reflect.Type
	// optional, to replace default config
	CtxMarshaler CtxMarshaler
	// optional, max retry times before fail
	RetryTimes int
	// optional, retry interval
	RetryInterval func(times int) time.Duration
	// optional, determine whether the task will be cleaned immediately once succeeded
	CleanSucceeded bool
	// optional, determine whether the initialized task can still be scheduled after timeout
	InitTimeoutSensitive bool

	// for built-in task only
	builtin      bool
	taskID       uint64
	argument     interface{}
	loopInterval time.Duration

	// inner use
	key TaskKey
}

func (s *TaskDefinition) GetCtxMarshaler(config *Config) CtxMarshaler {
	if m := s.CtxMarshaler; m != nil {
		return m
	}
	return config.CtxMarshaler
}

func (s *TaskDefinition) GetRetryInterval(times int) time.Duration {
	if f := s.RetryInterval; f != nil {
		return f(times)
	}
	return defaultRetryInterval
}

func (s *TaskDefinition) init(key TaskKey) error {
	if s.Handler == nil {
		return ErrDefNilHandler
	}
	if s.builtin {
		if s.taskID == 0 {
			return ErrDefEmptyPrimaryKey
		}
		if s.loopInterval <= 0 {
			return ErrDefInvalidLoopInterval
		}
		if s.argument == nil {
			return ErrDefInvalidArgument
		}
	}
	s.key = key
	return nil
}
