package gta

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
)

type Config struct {
	// must provide, async task table name
	TableName string
	// must provide, task db factory
	DBFactory func() *gorm.DB

	// optional, context for the task mansger
	Context context.Context
	// optional, task slave db factory
	SlaveDBFactory func() *gorm.DB
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
	CheckCallback func(abnormalTasks []TaskModel)
	// optional, flag for dry run mode
	DryRun bool
	// optional, goroutine pool size for scheduling tasks
	PoolSize int

	cancelFunc context.CancelFunc
}

func (s *Config) init() error {
	if s.TableName == "" {
		return ErrConfigEmptyTable
	}
	if s.DBFactory == nil {
		return ErrConfigNilDBFactory
	}

	// default value for optional config
	if s.Context == nil {
		s.Context = defaultContext()
	}
	if s.SlaveDBFactory == nil {
		s.SlaveDBFactory = s.DBFactory
	}
	if s.LoggerFactory == nil {
		s.LoggerFactory = defaultLoggerFactory()
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
		s.CtxMarshaler = defaultCtxMarshaler{}
	}

	if s.CheckCallback == nil {
		s.CheckCallback = defaultCheckCallback(s.logger())
	}

	if s.PoolSize <= 0 {
		s.PoolSize = defaultPoolSize
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

func (s *Config) load(options ...Option) *Config {
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Config) logger() Logger {
	return s.LoggerFactory(s.Context)
}

func (s *Config) done() <-chan struct{} {
	return s.Context.Done()
}

func (s *Config) cancel() {
	s.cancelFunc()
}

type Logger interface {
	Printf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type CtxMarshaler interface {
	MarshalCtx(ctx context.Context) ([]byte, error)
	UnmarshalCtx(bytes []byte) (context.Context, error)
}

// Option represents the optional function.
type Option func(c *Config)

// WithConfig accepts the whole config.
func WithConfig(config Config) Option {
	return func(c *Config) { *c = config }
}

func WithContext(ctx context.Context) Option {
	return func(c *Config) { c.Context = ctx }
}

func WithSlaveDBFactory(f func() *gorm.DB) Option {
	return func(c *Config) { c.SlaveDBFactory = f }
}

func WithLoggerFactory(f func(ctx context.Context) Logger) Option {
	return func(c *Config) { c.LoggerFactory = f }
}

func WithStorageTimeout(d time.Duration) Option {
	return func(c *Config) { c.StorageTimeout = d }
}

func WithInitializedTimeout(d time.Duration) Option {
	return func(c *Config) { c.InitializedTimeout = d }
}

func WithRunningTimeout(d time.Duration) Option {
	return func(c *Config) { c.RunningTimeout = d }
}

func WithWaitTimeout(d time.Duration) Option {
	return func(c *Config) { c.WaitTimeout = d }
}

func WithScanInterval(d time.Duration) Option {
	return func(c *Config) { c.ScanInterval = d }
}

func WithInstantScanInterval(d time.Duration) Option {
	return func(c *Config) { c.InstantScanInvertal = d }
}

func WithCtxMarshaler(m CtxMarshaler) Option {
	return func(c *Config) { c.CtxMarshaler = m }
}

func WithCheckCallback(f func(abnormalTasks []TaskModel)) Option {
	return func(c *Config) { c.CheckCallback = f }
}

func WithDryRun(flag bool) Option {
	return func(c *Config) { c.DryRun = flag }
}

func WithPoolSize(size int) Option {
	return func(c *Config) { c.PoolSize = size }
}
