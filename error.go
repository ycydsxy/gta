package gta

import "errors"

var (
	ErrNotUpdated   = errors.New("not updated")
	ErrUnexpected   = errors.New("unexpected")
	ErrTaskNotFound = errors.New("task not found")

	// config
	ErrConfigEmptyTable                 = errors.New("config table is empty")
	ErrConfigNilDBFactory               = errors.New("config db factory is nil")
	ErrConfigInvalidRunningTimeout      = errors.New("config running timeout is invalid")
	ErrConfigInvalidInitializeTimeout   = errors.New("config initialize timeout is invalid")
	ErrConfigInvalidScanInterval        = errors.New("config scan interval is invalid")
	ErrConfigInvalidInstantScanInterval = errors.New("config instant scan interval is invalid")

	// definition
	ErrDefNilHandler          = errors.New("definition handler is nil")
	ErrDefEmptyPrimaryKey     = errors.New("definition primary key is empty")
	ErrDefInvalidLoopInterval = errors.New("definition loop interval is invalid")
	ErrDefInvalidArgument     = errors.New("definition argument is invalid")
)
