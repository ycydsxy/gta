package gta

import (
	"context"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

var (
	defaultTableName = "default_table_name"
	defaultDBFactory = func() *gorm.DB { return nil }
)

type emptyCtxMarshaler struct{}

func (s emptyCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (s emptyCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	return context.Background(), nil
}

func emptyContext() context.Context {
	return context.Background()
}

func emptyLoggerFactory() func(ctx context.Context) Logger {
	return func(ctx context.Context) Logger {
		return logrus.NewEntry(logrus.New())
	}
}

func emptyCheckCallback(logger Logger) func(abnormalTasks []TaskModel) {
	return func(abnormalTasks []TaskModel) {
		if len(abnormalTasks) == 0 {
			return
		}
		logger.Errorf("[emptyCheckCallback] abnormal tasks found, total[%v]", len(abnormalTasks))
		for _, at := range abnormalTasks {
			logger.Warnf("[emptyCheckCallback] abnormal task found, id[%v], task_key[%v], task_status[%v]", at.ID, at.TaskKey, at.TaskStatus)
		}
	}
}
