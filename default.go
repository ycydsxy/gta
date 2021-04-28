package gta

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/panjf2000/ants/v2"
	"github.com/sirupsen/logrus"
)

var (
	defaultTableName           = "default_table_name"
	defaultDBFactory           = func() *gorm.DB { return nil }
	defaultStorageTimeout      = 7 * 24 * time.Hour
	defaultWaitTimeout         = 0 * time.Second
	defaultScanInterval        = 5 * time.Second
	defaultInstantScanInvertal = 100 * time.Millisecond
	defaultRunningTimeout      = 30 * time.Minute
	defaultInitializedTimeout  = 5 * time.Minute
	defaultPoolSize            = ants.DefaultAntsPoolSize
	defaultRetryInterval       = time.Second
)

type defaultCtxMarshaler struct{}

func (s defaultCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (s defaultCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	return context.Background(), nil
}

func defaultContext() context.Context {
	return context.Background()
}

func defaultLoggerFactory() func(ctx context.Context) Logger {
	return func(ctx context.Context) Logger { return logrus.NewEntry(logrus.New()) }
}

func defaultCheckCallback(logger Logger) func(abnormalTasks []TaskModel) {
	return func(abnormalTasks []TaskModel) {
		if len(abnormalTasks) == 0 {
			return
		}
		logger.Errorf("[defaultCheckCallback] abnormal tasks found, total[%v]", len(abnormalTasks))
		for _, at := range abnormalTasks {
			logger.Warnf("[defaultCheckCallback] abnormal task found, id[%v], task_key[%v], task_status[%v]", at.ID, at.TaskKey, at.TaskStatus)
		}
	}
}
