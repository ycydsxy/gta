package gta

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type taskAssembler interface {
	AssembleTask(ctxIn context.Context, taskDef *TaskDefinition, arg interface{}) (task *Task, err error)
	DisassembleTask(taskDef *TaskDefinition, task *Task) (context.Context, interface{}, error)
}

type taskAssemblerImp struct {
	config *TaskConfig
}

func (s *taskAssemblerImp) AssembleTask(ctxIn context.Context, taskDef *TaskDefinition, arg interface{}) (task *Task, err error) {
	// check if arg is valid
	if argTExpected := taskDef.ArgType; argTExpected != nil {
		argVActual := reflect.ValueOf(arg)
		if argVActual.IsValid() && argVActual.Type() != argTExpected {
			return nil, fmt.Errorf("arg type mismatch: %s expected, %T passed in", argTExpected, arg)
		}
	}

	argBytes, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("get argBytes failed, err: %w", err)
	}

	ctxBytes, err := taskDef.ctxMarshaler(s.config).MarshalCtx(ctxIn)
	if err != nil {
		return nil, fmt.Errorf("get ctxBytes failed, err: %w", err)
	}
	task = &Task{
		ID:         taskDef.taskID,
		TaskKey:    taskDef.key,
		TaskStatus: TaskStatusUnKnown,
		Context:    ctxBytes,
		Argument:   argBytes,
	}
	return task, nil
}

func (s *taskAssemblerImp) DisassembleTask(taskDef *TaskDefinition, task *Task) (context.Context, interface{}, error) {
	ctxIn, err := taskDef.ctxMarshaler(s.config).UnmarshalCtx(task.Context)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshal task context error: %w", err)
	}

	var argP interface{}
	if t := taskDef.ArgType; t != nil {
		argP = reflect.New(t).Interface()
	} else {
		var argI interface{}
		argP = &argI
	}
	if err := json.Unmarshal(task.Argument, argP); err != nil {
		return nil, nil, fmt.Errorf("unmarshal arg error: %w", err)
	}
	argument := reflect.ValueOf(argP).Elem().Interface()

	return ctxIn, argument, nil
}
