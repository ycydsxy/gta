package gta

import "errors"

var (
	// ErrZeroRowsAffected represents zero rows affected in a database operation.
	ErrZeroRowsAffected = errors.New("zero rows affected")
	// ErrUnexpected represents unexpected error occurred.
	ErrUnexpected = errors.New("unexpected")
	// ErrTaskNotFound represents certain task not found.
	ErrTaskNotFound = errors.New("task not found")

	// ErrOption represents option is invalid.
	ErrOption = errors.New("option invalid")

	// ErrDefNilHandler represents Handler in the task definition is nil.
	ErrDefNilHandler = errors.New("definition handler is nil")
	// ErrDefEmptyPrimaryKey represents primaryKey in the task definition is empty.
	ErrDefEmptyPrimaryKey = errors.New("definition primary key is empty")
	// ErrDefInvalidLoopInterval represents loopInterval in the task definition is invalid.
	ErrDefInvalidLoopInterval = errors.New("definition loop interval is invalid")
	// ErrDefInvalidArgument represents argument in the task definition is invalid.
	ErrDefInvalidArgument = errors.New("definition argument is invalid")
)
