package gta

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/panjf2000/ants/v2"
)

const (
	defaultStorageTimeout      = 7 * 24 * time.Hour
	defaultWaitTimeout         = 0 * time.Second
	defaultScanInterval        = 5 * time.Second
	defaultInstantScanInvertal = 100 * time.Millisecond
	defaultRunningTimeout      = 30 * time.Minute
	defaultInitializedTimeout  = 5 * time.Minute
	defaultPoolSize            = ants.DefaultAntsPoolSize
)

type CtxMarshaler interface {
	MarshalCtx(ctx context.Context) ([]byte, error)
	UnmarshalCtx(bytes []byte) (context.Context, error)
}

type Logger interface {
	Printf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type Config struct {
	// must provide, async task table name
	TableName string
	// must provide, task db factory
	DBFactory func() *gorm.DB
	// optional, task slave db factory
	SlaveDBFactory func() *gorm.DB
	// optional, logger factory
	LoggerFactory func(ctx context.Context) Logger
	// optional, context for the task mansger
	Context context.Context
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
	if s.SlaveDBFactory == nil {
		s.SlaveDBFactory = s.DBFactory
	}
	if s.LoggerFactory == nil {
		s.LoggerFactory = emptyLoggerFactory()
	}
	if s.Context == nil {
		s.Context = emptyContext()
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
		s.CtxMarshaler = emptyCtxMarshaler{}
	}

	if s.CheckCallback == nil {
		s.CheckCallback = emptyCheckCallback(s.logger())
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

// Option represents the optional function.
type Option func(opts *Config)

// WithConfig accepts the whole config.
func WithConfig(config Config) Option {
	return func(opts *Config) { *opts = config }
}
