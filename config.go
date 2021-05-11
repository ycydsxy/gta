package gta

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// TaskConfig contains all options of a TaskManager.
type TaskConfig struct {
	// must provide, db for async task table
	DB *gorm.DB
	// must provide, async task table name
	Table string

	// optional, context for the task mansger
	Context context.Context
	// optional, logger factory
	LoggerFactory func(ctx context.Context) Logger
	// optional, determine when a normal task can be cleaned
	StorageTimeout time.Duration
	// optional, determine whether a initialized task is abnormal
	InitializedTimeout time.Duration
	// optional, determine whether a running task is abnormal
	RunningTimeout time.Duration
	// optional, wait timeout in Stop() process
	WaitTimeout time.Duration
	// optional, scan interval
	ScanInterval time.Duration
	// optional, instant scan interval
	InstantScanInvertal time.Duration
	// optional, context marshaler to store or recover a context
	CtxMarshaler CtxMarshaler
	// optional, callback function for abnormal tasks
	CheckCallback func(logger Logger, abnormalTasks []Task)
	// optional, flag for dry run mode
	DryRun bool
	// optional, goroutine pool size for scheduling tasks
	PoolSize int

	// inner use
	taskRegister taskRegister
	cancelFunc   context.CancelFunc
}

func (s *TaskConfig) init() error {
	if s.DB == nil {
		return ErrConfigNilDB
	}
	if s.Table == "" {
		return ErrConfigEmptyTable
	}

	// default value for optional config
	if s.Context == nil {
		s.Context = defaultContextFactory()
	}
	if s.LoggerFactory == nil {
		s.LoggerFactory = defaultLoggerFactory
	}
	if s.StorageTimeout <= 0 {
		s.StorageTimeout = defaultStorageTimeout
	}
	if s.WaitTimeout <= 0 {
		s.WaitTimeout = defaultWaitTimeout
	}
	if s.ScanInterval <= 0 {
		s.ScanInterval = defaultScanInterval
	}
	if s.InstantScanInvertal <= 0 {
		s.InstantScanInvertal = defaultInstantScanInvertal
	}
	if s.RunningTimeout <= 0 {
		s.RunningTimeout = defaultRunningTimeout
	}
	if s.InitializedTimeout <= 0 {
		s.InitializedTimeout = defaultInitializedTimeout
	}
	if s.CtxMarshaler == nil {
		s.CtxMarshaler = &defaultCtxMarshaler{}
	}
	if s.CheckCallback == nil {
		s.CheckCallback = defaultCheckCallback
	}
	if s.PoolSize <= 0 {
		s.PoolSize = defaultPoolSize
	}
	if s.taskRegister == nil {
		s.taskRegister = &taskRegisterImp{}
	}

	// check
	if s.RunningTimeout > s.StorageTimeout {
		return ErrConfigInvalidRunningTimeout
	}
	if s.InitializedTimeout > s.StorageTimeout {
		return ErrConfigInvalidInitializeTimeout
	}
	if s.ScanInterval > s.StorageTimeout || s.ScanInterval > s.InitializedTimeout || s.ScanInterval > s.RunningTimeout {
		return ErrConfigInvalidScanInterval
	}
	if s.InstantScanInvertal > s.ScanInterval {
		return ErrConfigInvalidInstantScanInterval
	}

	// generate context with cancel
	s.Context, s.cancelFunc = context.WithCancel(s.Context)

	return nil
}

func (s *TaskConfig) load(options ...Option) *TaskConfig {
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *TaskConfig) logger() Logger {
	return s.LoggerFactory(s.Context)
}

func (s *TaskConfig) done() <-chan struct{} {
	return s.Context.Done()
}

func (s *TaskConfig) cancel() {
	s.cancelFunc()
}

func newConfig(db *gorm.DB, table string, options ...Option) (*TaskConfig, error) {
	c := (&TaskConfig{}).load(options...).load(withDB(db)).load(withTable(table))
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

// Logger is a logging interface for logging necessary messages.
type Logger interface {
	Printf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// CtxMarshaler is used to marshal or unmarshal context.
type CtxMarshaler interface {
	MarshalCtx(ctx context.Context) ([]byte, error)
	UnmarshalCtx(bytes []byte) (context.Context, error)
}

// Option represents the optional function.
type Option func(c *TaskConfig)

// WithConfig set the whole config.
func WithConfig(config TaskConfig) Option {
	return func(c *TaskConfig) { *c = config }
}

// WithContext set the Context option.
func WithContext(ctx context.Context) Option {
	return func(c *TaskConfig) { c.Context = ctx }
}

// WithLoggerFactory set the LoggerFactory option.
func WithLoggerFactory(f func(ctx context.Context) Logger) Option {
	return func(c *TaskConfig) { c.LoggerFactory = f }
}

// WithStorageTimeout set the StorageTimeout option.
func WithStorageTimeout(d time.Duration) Option {
	return func(c *TaskConfig) { c.StorageTimeout = d }
}

// WithInitializedTimeout set the InitializedTimeout option.
func WithInitializedTimeout(d time.Duration) Option {
	return func(c *TaskConfig) { c.InitializedTimeout = d }
}

// WithRunningTimeout set the RunningTimeout option.
func WithRunningTimeout(d time.Duration) Option {
	return func(c *TaskConfig) { c.RunningTimeout = d }
}

// WithWaitTimeout set the WaitTimeout option.
func WithWaitTimeout(d time.Duration) Option {
	return func(c *TaskConfig) { c.WaitTimeout = d }
}

// WithScanInterval set the ScanInterval option.
func WithScanInterval(d time.Duration) Option {
	return func(c *TaskConfig) { c.ScanInterval = d }
}

// WithInstantScanInterval set the InstantScanInvertal option.
func WithInstantScanInterval(d time.Duration) Option {
	return func(c *TaskConfig) { c.InstantScanInvertal = d }
}

// WithCtxMarshaler set the CtxMarshaler option.
func WithCtxMarshaler(m CtxMarshaler) Option {
	return func(c *TaskConfig) { c.CtxMarshaler = m }
}

// WithCheckCallback set the CheckCallback option.
func WithCheckCallback(f func(logger Logger, abnormalTasks []Task)) Option {
	return func(c *TaskConfig) { c.CheckCallback = f }
}

// WithDryRun set the DryRun option.
func WithDryRun(flag bool) Option {
	return func(c *TaskConfig) { c.DryRun = flag }
}

// WithPoolSize set the PoolSize option.
func WithPoolSize(size int) Option {
	return func(c *TaskConfig) { c.PoolSize = size }
}

func withDB(db *gorm.DB) Option {
	return func(c *TaskConfig) { c.DB = db }
}

func withTable(table string) Option {
	return func(c *TaskConfig) { c.Table = table }
}

func withTaskRegister(tr taskRegister) Option {
	return func(c *TaskConfig) {
		c.taskRegister = tr
	}
}
