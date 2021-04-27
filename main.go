package gta

import (
	"context"

	"github.com/jinzhu/gorm"
)

var (
	defaultManager = NewTaskManager(defaultDBFactory, defaultTableName)
)

// Start initializes the overall configuration. It starts a scan process asynchronously after everything is ready.
//
// The task table name and database factory method must be provided in the initialize process because this tool relies
// heavily on the database. For more information about the table schema, please refer to 'model.go'.
func Start(dbFactory func() *gorm.DB, tableName string, options ...Option) {
	c := Config{DBFactory: dbFactory, TableName: tableName}
	if err := defaultManager.tc.load(WithConfig(c)).load(options...).init(); err != nil {
		panic(err)
	}
	defaultManager.Start()
}

func StartWithConfig(c Config) {
	if err := defaultManager.tc.load(WithConfig(c)).init(); err != nil {
		panic(err)
	}
	defaultManager.Start()
}

func Register(key TaskKey, definition TaskDefinition) {
	defaultManager.Register(key, definition)
}

func Run(ctx context.Context, key TaskKey, arg interface{}) error {
	return defaultManager.Run(ctx, key, arg)
}

func RunWithTx(tx *gorm.DB, ctx context.Context, key TaskKey, arg interface{}) error {
	return defaultManager.RunWithTx(tx, ctx, key, arg)
}

func Transaction(fc func(tx *gorm.DB) error) (err error) {
	return defaultManager.Transaction(fc)
}

func Stop(wait bool) {
	defaultManager.Stop(wait)
}

func Wait() {
	defaultManager.Wait()
}

/* Following are DevOps APIs */

// ForceRerunTask changes the specific task to 'initialized'.
func ForceRerunTask(taskID uint64, status TaskStatus) error {
	return defaultManager.ForceRerunTask(taskID, status)
}

// ForceRerunTasks changes specific tasks to 'initialized'.
func ForceRerunTasks(taskIDs []uint64, status TaskStatus) (int64, error) {
	return defaultManager.ForceRerunTasks(taskIDs, status)
}

// QueryUnsuccessfulTasks checks initialized, running or failed tasks.
func QueryUnsuccessfulTasks() ([]TaskModel, error) {
	return defaultManager.QueryUnsuccessfulTasks()
}
