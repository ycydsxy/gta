package gta

import "errors"

var (
	// ErrZeroRowsAffected represents zero rows affected in a database operation.
	ErrZeroRowsAffected = errors.New("zero rows affected")
	// ErrUnexpected represents unexpected error occurred.
	ErrUnexpected = errors.New("unexpected")
	// ErrTaskNotFound represents certain task not found.
	ErrTaskNotFound = errors.New("task not found")

	// config
	// ErrConfigEmptyTable represents TableName in the config is empty.
	ErrConfigEmptyTable = errors.New("config table is empty")
	// ErrConfigNilDB represents DB in the config is nil.
	ErrConfigNilDB = errors.New("config db is nil")
	// ErrConfigInvalidRunningTimeout represents RunningTimeout in the config is invalid.
	ErrConfigInvalidRunningTimeout = errors.New("config running timeout is invalid")
	// ErrConfigInvalidInitializeTimeout represents InitializeTimeout in the config is invalid.
	ErrConfigInvalidInitializeTimeout = errors.New("config initialize timeout is invalid")
	// ErrConfigInvalidScanInterval represents ScanInterval in the config is invalid.
	ErrConfigInvalidScanInterval = errors.New("config scan interval is invalid")
	// ErrConfigInvalidInstantScanInterval represents InstantScanInterval in the config is invalid.
	ErrConfigInvalidInstantScanInterval = errors.New("config instant scan interval is invalid")

	// definition
	// ErrDefNilHandler represents Handler in the task definition is nil.
	ErrDefNilHandler = errors.New("definition handler is nil")
	// ErrDefEmptyPrimaryKey represents primaryKey in the task definition is empty.
	ErrDefEmptyPrimaryKey = errors.New("definition primary key is empty")
	// ErrDefInvalidLoopInterval represents loopInterval in the task definition is invalid.
	ErrDefInvalidLoopInterval = errors.New("definition loop interval is invalid")
	// ErrDefInvalidArgument represents argument in the task definition is invalid.
	ErrDefInvalidArgument = errors.New("definition argument is invalid")
)
