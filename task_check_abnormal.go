package gta

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

const (
	taskCheckAbnormal   TaskKey = "builtin:check_abnormal_task"
	taskCheckAbnormalID uint64  = 10000
)

type checkAbnormalTaskReq struct {
	StorageTimeout     time.Duration `json:"storage_timeout"`
	RunningTimeout     time.Duration `json:"running_timeout"`
	InitializedTimeout time.Duration `json:"initialized_timeout"`
}

func registerCheckAbnormalTask(tm *TaskManager) {
	tc := tm.tc
	tm.Register(taskCheckAbnormal, TaskDefinition{
		Handler: checkAbnormalHandler(tm),
		ArgType: reflect.TypeOf(checkAbnormalTaskReq{}),
		builtin: true,
		taskID:  taskCheckAbnormalID,
		argument: checkAbnormalTaskReq{
			StorageTimeout:     tc.StorageTimeout,
			RunningTimeout:     tc.RunningTimeout,
			InitializedTimeout: tc.InitializedTimeout,
		},
		loopInterval: time.Duration(
			minInt64(int64(tc.InitializedTimeout)/2, int64(tc.RunningTimeout)/2, int64(tc.ScanInterval)*15),
		),
	})
}

func checkAbnormalHandler(tm *TaskManager) TaskHandler {
	tc := tm.tc
	return func(ctx context.Context, arg interface{}) (err error) {
		req := arg.(checkAbnormalTaskReq)
		abnormalRunning, err := tm.tdal.GetSliceByOffsetsAndStatus(tc.DB, req.StorageTimeout,
			req.RunningTimeout, taskStatusRunning)
		if err != nil {
			return fmt.Errorf("check abnormal running failed")
		}
		abnormalInitilized, err := tm.tdal.GetSliceByOffsetsAndStatus(tc.DB, req.StorageTimeout,
			req.InitializedTimeout, taskStatusInitialized)
		if err != nil {
			return fmt.Errorf("check abnormal running failed")
		}

		builtinKeys := tm.tr.GetBuiltInKeys()
		builtinSet := make(map[TaskKey]struct{}, len(builtinKeys))
		for _, bk := range builtinKeys {
			builtinSet[bk] = struct{}{}
		}

		abnormalTasks := make([]Task, 0, len(abnormalRunning)+len(abnormalInitilized))
		for _, t := range abnormalRunning {
			if _, ok := builtinSet[t.TaskKey]; ok {
				continue
			}
			abnormalTasks = append(abnormalTasks, t)
		}
		for _, t := range abnormalInitilized {
			if _, ok := builtinSet[t.TaskKey]; ok {
				continue
			}
			abnormalTasks = append(abnormalTasks, t)
		}

		tc.CheckCallback(abnormalTasks)
		return nil
	}
}
