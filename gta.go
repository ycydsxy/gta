package gta

import (
	"context"

	"gorm.io/gorm"
)

var (
	defaultTaskManager = &TaskManager{tr: &taskRegisterImp{}}
)

// StartWithOptions configures the default task manager and starts it. This function should be called before any other
// function is called.
func StartWithOptions(db *gorm.DB, table string, options ...Option) {
	opts := make([]Option, len(options))
	copy(opts, options)
	opts = append(opts, withTaskRegister(defaultTaskManager.tr))
	defaultTaskManager = NewTaskManager(db, table, opts...)
	defaultTaskManager.Start()
}

// Register binds a task definition to a certain task key.
func Register(key TaskKey, definition TaskDefinition) {
	defaultTaskManager.Register(key, definition)
}

// Run provides the ability to asynchronously run a registered task reliably.
func Run(ctx context.Context, key TaskKey, arg interface{}) error {
	return defaultTaskManager.Run(ctx, key, arg)
}

// RunWithTx makes it possible to create a task along with other database operations in the same transaction.
func RunWithTx(tx *gorm.DB, ctx context.Context, key TaskKey, arg interface{}) error {
	return defaultTaskManager.RunWithTx(tx, ctx, key, arg)
}

// Transaction wraps the 'Transaction' function of *gorm.DB
func Transaction(fc func(tx *gorm.DB) error) (err error) {
	return defaultTaskManager.Transaction(fc)
}

// Stop provides the ability to gracefully stop current running tasks.
func Stop(wait bool) {
	defaultTaskManager.Stop(wait)
}

// Wait blocks the current goroutine and waits for a termination signal.
func Wait() {
	defaultTaskManager.Wait()
}

// DefaultManager returns the default task manager.
func DefaultManager() *TaskManager {
	return defaultTaskManager
}
