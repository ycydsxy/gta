package gta

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/panjf2000/ants/v2"
)

func NewTaskManager(dbFactory func() *gorm.DB, tableName string, options ...Option) *TaskManager {
	tc := &Config{TableName: tableName, DBFactory: dbFactory}
	if err := tc.load(options...).init(); err != nil {
		panic(err)
	}
	tm := new(TaskManager)
	tm.tc = tc
	tm.tr = &taskRegisterImp{}
	tm.tdal = &taskDALImp{config: tc}
	tm.tass = &taskAssemblerImp{config: tc}
	pool, err := ants.NewPool(tc.PoolSize, ants.WithLogger(tc.logger()), ants.WithNonblocking(true))
	if err != nil {
		panic(err)
	}
	tm.tsch = &taskSchedulerImp{config: tc, register: tm.tr, dal: tm.tdal, assembler: tm.tass, pool: pool}
	tm.tmon = &taskMonitorImp{config: tc, register: tm.tr, dal: tm.tdal, assembler: tm.tass}
	tm.tscn = &taskScannerImp{config: tc, register: tm.tr, dal: tm.tdal, scheduler: tm.tsch}

	return tm
}

type TaskManager struct {
	tc   *Config
	tr   taskRegister
	tass taskAssembler
	tsch taskScheduler
	tdal taskDAL
	tmon taskMonitor
	tscn taskScanner

	startOnce sync.Once
	stopOnce  sync.Once
}

func (s *TaskManager) Start() {
	s.startOnce.Do(func() {
		if s.tc.DryRun {
			// don't start scan and monitor process in dry run mode
			return
		}
		s.registerBuiltinTasks()
		s.tscn.GoScanAndSchedule()
		s.tmon.GoMonitorBuiltinTasks()
		time.Sleep(time.Second)
	})
}

// Register binds a task definition to a certain task key. Tasks of same type usually have the same task key.
//
// Task key is a unique ID for a set of tasks with same definition. Task handler should be idempotent because a task may
// be scheduled more than once in some cases.
//
// Handler must be provided in the task definition. It would be better to provide the argument type additionally, unless
// you want to use the default argument type(i.e. map[string]interface{} for struct) inside the handler.
func (s *TaskManager) Register(key TaskKey, definition TaskDefinition) {
	if err := s.tr.Register(key, definition); err != nil {
		panic(err)
	}
}

// Run provides the ability to asynchronously run a registered task reliably. It's an alternative to using 'go func(
// ){}' when you need to care about the ultimate success of a task.

// An error is returned when the task creating process failed, otherwise, the task will be scheduled asynchronously
// later. If error or panic occurs in the running process, it will be rescheduled according to the config. If the
// retry times exceeds the maximum config value, the task is marked 'failed' in the database with error logs recorded.
// In these cases, maybe a manual operation is essential.
//
// The context passed in should be consistent with the 'CtxMarshaler' config defined in the overall configuration or the
// task definition.
func (s *TaskManager) Run(ctx context.Context, key TaskKey, arg interface{}) error {
	return s.Transaction(func(tx *gorm.DB) error { return s.RunWithTx(tx, ctx, key, arg) })
}

// RunWithTx makes it possible to create a task along with other database operations in the same transaction. The task
// will be scheduled if the transaction is committed successfully, or canceled if the transaction is rolled backs. Thus,
// this is a simple implement for BASE that can be used in distributed transaction situations.
//
// The task will be scheduled immediately after the transaction is committed if you use the builtin 'Transaction'
// function below. Otherwise, it will be scheduled later in the scan process.
//
// You can create more than one task in a single transaction, like this:
//
// _ = Transaction(func(tx *gorm.DB) error {
//		if err:= doSomething(); err != nil{ // do something
//			return err
//		}
//
// 		if err := RunWithTx(); err != nil { // task1
// 			return err
// 		}
//
// 		if err := RunWithTx(); err != nil { // task2
// 			return err
// 		}
// 		return nil
// })
//
func (s *TaskManager) RunWithTx(tx *gorm.DB, ctx context.Context, key TaskKey, arg interface{}) error {
	return s.tsch.CreateTask(tx, ctx, key, arg)
}

// Transaction wraps the 'Transaction' function of *gorm.DB, providing the ability to schedule the tasks created inside
// once the transaction is committed successfully.
func (s *TaskManager) Transaction(fc func(tx *gorm.DB) error) (err error) {
	return s.tsch.Transaction(fc)
}

// Stop provides the ability to gracefully stop current running tasks. If you cannot tolerate task failure or loss in
// cases when a termination signal is received or the pod is migrated, it would be better to explicitly call this
// function before the main process exits. Otherwise, these tasks are easily to be killed and will be reported by
// abnormal task check process later.
func (s *TaskManager) Stop(wait bool) {
	s.stopOnce.Do(func() {
		if s.tc.DryRun {
			return
		}
		s.tc.cancel() // send global cancel signal
		s.tsch.Stop(wait)
	})
}

// Wait blocks the current goroutine and waits for a termination signal. Stop() will be called after the termination
// signal is received. Maybe this function is useless, because the main function is always blocked by others, like a
// gin server.
func (s *TaskManager) Wait() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	select {
	case <-ch:
		Stop(false)
		return
	}
}

// ForceRerunTask changes the specific task to 'initialized'.
func (s *TaskManager) ForceRerunTask(taskID uint64, status TaskStatus) error {
	rows, err := s.ForceRerunTasks([]uint64{taskID}, status)
	if err != nil {
		return err
	} else if rows != 1 {
		return ErrNotUpdated
	}
	return nil
}

// ForceRerunTasks changes specific tasks to 'initialized'.
func (s *TaskManager) ForceRerunTasks(taskIDs []uint64, status TaskStatus) (int64, error) {
	return s.tdal.ForceUpdateTaskStatusByIDs(s.tc.DBFactory(), taskIDs, status, taskStatusInitialized)
}

// QueryUnsuccessfulTasks checks initialized, running or failed tasks.
func (s *TaskManager) QueryUnsuccessfulTasks() ([]TaskModel, error) {
	return s.tdal.GetSliceExcludeSucceeded(s.tc.DBFactory(), s.tr.GetBuiltInKeys())
}

func (s *TaskManager) registerBuiltinTasks() {
	registerCleanUpTask(s)
	registerCheckAbnormalTask(s)
}