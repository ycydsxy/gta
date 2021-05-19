package gta

import (
	"context"
	"fmt"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func newOptions(db *gorm.DB, table string, opts ...Option) (*options, error) {
	options, err := newDefaultOptions().apply(append(optionGroup{withDB(db), withTable(table)}, opts...))
	if err != nil {
		return nil, err
	}
	return options, nil
}

// options contains all options of a TaskManager.
type options struct {
	// must provide, db for async task table
	db *gorm.DB
	// must provide, async task table name
	table string

	// optional, context for the task mansger
	context context.Context
	// optional, logger factory
	loggerFactory func(ctx context.Context) Logger
	// optional, grouped time options
	groupedTimeOptions
	// optional, wait timeout in Stop() process
	waitTimeout time.Duration
	// optional, context marshaler to store or recover a context
	ctxMarshaler CtxMarshaler
	// optional, callback function for abnormal tasks
	checkCallback func(logger Logger, abnormalTasks []Task)
	// optional, flag for dry run mode
	dryRun bool
	// optional, goroutine pool size for scheduling tasks
	poolSize int

	// optional, task register
	taskRegister taskRegister
	// global cancel function
	cancelFunc context.CancelFunc
}

func (s *options) apply(opts ...Option) (*options, error) {
	for _, opt := range opts {
		opt.apply(s)
	}
	for _, opt := range opts {
		if err := opt.verify(s); err != nil {
			return s, err
		}
	}
	return s, nil
}

func (s *options) logger() Logger {
	return s.loggerFactory(s.context)
}

func (s *options) done() <-chan struct{} {
	return s.context.Done()
}

func (s *options) cancel() {
	s.cancelFunc()
}

func (s *options) getDB() *gorm.DB {
	return s.db
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

type groupedTimeOptions struct {
	// optional, determine when a normal task can be cleaned
	storageTimeout time.Duration
	// optional, determine whether a initialized task is abnormal
	initializedTimeout time.Duration
	// optional, determine whether a running task is abnormal
	runningTimeout time.Duration
	// optional, scan interval
	scanInterval time.Duration
	// optional, instant scan interval
	instantScanInterval time.Duration
}

func (t groupedTimeOptions) verify() error {
	if t.storageTimeout <= 0 || t.initializedTimeout <= 0 || t.runningTimeout <= 0 || t.scanInterval <= 0 || t.instantScanInterval <= 0 {
		return fmt.Errorf("%w: groupedTimeOptions #1", ErrOption)
	}
	if t.runningTimeout > t.storageTimeout {
		return fmt.Errorf("%w: groupedTimeOptions #2", ErrOption)
	}
	if t.initializedTimeout > t.storageTimeout {
		return fmt.Errorf("%w: groupedTimeOptions #3", ErrOption)
	}
	if t.scanInterval > t.storageTimeout || t.scanInterval > t.initializedTimeout || t.scanInterval > t.runningTimeout {
		return fmt.Errorf("%w: groupedTimeOptions #4", ErrOption)
	}
	if t.instantScanInterval > t.scanInterval {
		return fmt.Errorf("%w: groupedTimeOptions #5", ErrOption)
	}
	return nil
}

// Option is a interface.
type Option interface {
	apply(opts *options)
	verify(opts *options) error
}

type option struct {
	applyFunc  func(opts *options)
	verifyFunc func(opts *options) error
}

func (o option) apply(opts *options) {
	o.applyFunc(opts)
}

func (o option) verify(opts *options) error {
	if o.verifyFunc == nil {
		return nil
	}
	return o.verifyFunc(opts)
}

type optionGroup []Option

func (g optionGroup) apply(opts *options) {
	for _, opt := range g {
		opt.apply(opts)
	}
}

func (g optionGroup) verify(opts *options) error {
	for _, opt := range g {
		if err := opt.verify(opts); err != nil {
			return err
		}
	}
	return nil
}

// WithContext set the context option.
func WithContext(ctx context.Context) Option {
	return &option{
		applyFunc: func(opts *options) { opts.context, opts.cancelFunc = context.WithCancel(ctx) },
	}
}

// WithLoggerFactory set the loggerFactory option.
func WithLoggerFactory(f func(ctx context.Context) Logger) Option {
	return &option{
		applyFunc: func(opts *options) { opts.loggerFactory = f },
		verifyFunc: func(opts *options) error {
			if opts.loggerFactory == nil {
				return fmt.Errorf("%w: loggerFactory", ErrOption)
			}
			return nil
		},
	}
}

// WithStorageTimeout set the storageTimeout option.
func WithStorageTimeout(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.storageTimeout = d },
		verifyFunc: func(opts *options) error {
			return opts.groupedTimeOptions.verify()
		},
	}
}

// WithInitializedTimeout set the initializedTimeout option.
func WithInitializedTimeout(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.initializedTimeout = d },
		verifyFunc: func(opts *options) error {
			return opts.groupedTimeOptions.verify()
		},
	}
}

// WithRunningTimeout set the runningTimeout option.
func WithRunningTimeout(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.runningTimeout = d },
		verifyFunc: func(opts *options) error {
			return opts.groupedTimeOptions.verify()
		},
	}
}

// WithScanInterval set the scanInterval option.
func WithScanInterval(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.scanInterval = d },
		verifyFunc: func(opts *options) error {
			return opts.groupedTimeOptions.verify()
		},
	}
}

// WithInstantScanInterval set the instantScanInterval option.
func WithInstantScanInterval(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.instantScanInterval = d },
		verifyFunc: func(opts *options) error {
			return opts.groupedTimeOptions.verify()
		},
	}
}

// WithWaitTimeout set the waitTimeout option.
func WithWaitTimeout(d time.Duration) Option {
	return &option{
		applyFunc: func(opts *options) { opts.waitTimeout = d },
		verifyFunc: func(opts *options) error {
			if opts.waitTimeout <= 0 {
				return fmt.Errorf("%w: waitTimeout", ErrOption)
			}
			return nil
		},
	}
}

// WithCtxMarshaler set the ctxMarshaler option.
func WithCtxMarshaler(m CtxMarshaler) Option {
	return &option{
		applyFunc: func(opts *options) { opts.ctxMarshaler = m },
		verifyFunc: func(opts *options) error {
			if opts.ctxMarshaler == nil {
				return fmt.Errorf("%w: ctxMarshaler", ErrOption)
			}
			return nil
		},
	}
}

// WithCheckCallback set the checkCallback option.
func WithCheckCallback(f func(logger Logger, abnormalTasks []Task)) Option {
	return &option{
		applyFunc: func(opts *options) { opts.checkCallback = f },
		verifyFunc: func(opts *options) error {
			if opts.checkCallback == nil {
				return fmt.Errorf("%w: checkCallback", ErrOption)
			}
			return nil
		},
	}
}

// WithDryRun set the dryRun option.
func WithDryRun(flag bool) Option {
	return &option{
		applyFunc: func(opts *options) { opts.dryRun = flag },
	}
}

// WithPoolSize set the poolSize option.
func WithPoolSize(size int) Option {
	return &option{
		applyFunc: func(opts *options) { opts.poolSize = size },
		verifyFunc: func(opts *options) error {
			if opts.poolSize <= 0 {
				return fmt.Errorf("%w: poolSize", ErrOption)
			}
			return nil
		},
	}
}

func withDB(db *gorm.DB) Option {
	return &option{
		applyFunc: func(opts *options) { opts.db = db },
		verifyFunc: func(opts *options) error {
			if opts.db == nil {
				return fmt.Errorf("%w: db", ErrOption)
			}
			return nil
		},
	}
}

func withTable(table string) Option {
	return &option{
		applyFunc: func(opts *options) { opts.table = table },
		verifyFunc: func(opts *options) error {
			if opts.table == "" {
				return fmt.Errorf("%w: table", ErrOption)
			}
			return nil
		},
	}
}

func withTaskRegister(tr taskRegister) Option {
	return &option{
		applyFunc: func(opts *options) { opts.taskRegister = tr },
		verifyFunc: func(opts *options) error {
			if opts.taskRegister == nil {
				return fmt.Errorf("%w: taskRegister", ErrOption)
			}
			return nil
		},
	}
}

// newDefaultOptions generate default options
func newDefaultOptions() *options {
	ctx, cancelFunc := context.WithCancel(defaultContextFactory())
	return &options{
		db:            nil,
		table:         "",
		context:       ctx,
		loggerFactory: defaultLoggerFactory,
		groupedTimeOptions: groupedTimeOptions{
			storageTimeout:      defaultStorageTimeout,
			initializedTimeout:  defaultInitializedTimeout,
			runningTimeout:      defaultRunningTimeout,
			scanInterval:        defaultScanInterval,
			instantScanInterval: defaultInstantScanInterval,
		},
		waitTimeout:   defaultWaitTimeout,
		ctxMarshaler:  &defaultCtxMarshaler{},
		checkCallback: defaultCheckCallback,
		dryRun:        false,
		poolSize:      defaultPoolSize,
		taskRegister:  &taskRegisterImp{},
		cancelFunc:    cancelFunc,
	}
}

var (
	defaultStorageTimeout      = time.Hour * 7 * 24
	defaultWaitTimeout         = time.Second * 0
	defaultScanInterval        = time.Second * 5
	defaultInstantScanInterval = time.Millisecond * 100
	defaultRunningTimeout      = time.Minute * 30
	defaultInitializedTimeout  = time.Minute * 5
	defaultPoolSize            = ants.DefaultAntsPoolSize
	defaultRetryInterval       = time.Second
)

type defaultCtxMarshaler struct{}

func (s *defaultCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (s *defaultCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	return context.Background(), nil
}

func defaultContextFactory() context.Context {
	return context.Background()
}

func defaultLoggerFactory(ctx context.Context) Logger {
	return logrus.NewEntry(logrus.New())
}

func defaultCheckCallback(logger Logger, abnormalTasks []Task) {
	if len(abnormalTasks) == 0 {
		return
	}
	logger.Errorf("[defaultCheckCallback] abnormal tasks found, total[%v]", len(abnormalTasks))
	for _, at := range abnormalTasks {
		logger.Warnf("[defaultCheckCallback] abnormal task found, id[%v], task_key[%v], task_status[%v]", at.ID, at.TaskKey, at.TaskStatus)
	}
}
