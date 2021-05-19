package gta

import (
	"context"
	"reflect"
	"time"
)

const (
	taskCleanUp   TaskKey = "builtin:clean_up"
	taskCleanUpID uint64  = 9999
)

type cleanUpReq struct {
	StorageTimeout time.Duration `json:"storage_timeout"`
}

func registerCleanUpTask(tm *TaskManager) {
	tm.Register(taskCleanUp, TaskDefinition{
		Handler:      cleanUpHandler(tm),
		ArgType:      reflect.TypeOf(cleanUpReq{}),
		builtin:      true,
		taskID:       taskCleanUpID,
		argument:     cleanUpReq{StorageTimeout: tm.storageTimeout},
		loopInterval: tm.storageTimeout / 2,
	})
}

func cleanUpHandler(tm *TaskManager) TaskHandler {
	return func(ctx context.Context, arg interface{}) (err error) {
		logger := tm.logger()
		storageTimeout := arg.(cleanUpReq).StorageTimeout
		rowsAffected, err := tm.tdal.DeleteSucceededByOffset(tm.getDB(), storageTimeout, tm.tr.GetBuiltInKeys())
		if err != nil {
			return err
		} else if rowsAffected > 0 {
			logger.Infof("[cleanUpHandler] task cleaned, storage timeout[%v], len[%v]", storageTimeout, rowsAffected)
		}
		return nil
	}
}
