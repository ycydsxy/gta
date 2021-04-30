package gta

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

var (
	defaultStorageTimeout      = time.Hour * 7 * 24
	defaultWaitTimeout         = time.Second * 0
	defaultScanInterval        = time.Second * 5
	defaultInstantScanInvertal = time.Millisecond * 100
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

func defaultContext() context.Context {
	return context.Background()
}

func defaultLoggerFactory() func(ctx context.Context) Logger {
	return func(ctx context.Context) Logger { return logrus.NewEntry(logrus.New()) }
}

func defaultCheckCallback(logger Logger) func(abnormalTasks []Task) {
	return func(abnormalTasks []Task) {
		if len(abnormalTasks) == 0 {
			return
		}
		logger.Errorf("[defaultCheckCallback] abnormal tasks found, total[%v]", len(abnormalTasks))
		for _, at := range abnormalTasks {
			logger.Warnf("[defaultCheckCallback] abnormal task found, id[%v], task_key[%v], task_status[%v]", at.ID, at.TaskKey, at.TaskStatus)
		}
	}
}
