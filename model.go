package gta

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

const (
	taskStatusUnKnown     TaskStatus = ""
	taskStatusInitialized TaskStatus = "initialized"
	taskStatusRunning     TaskStatus = "running"
	taskStatusSucceeded   TaskStatus = "succeeded"
	taskStatusFailed      TaskStatus = "failed"
)

type TaskKey string

type TaskStatus string

type TaskModel struct {
	ID         uint64
	TaskKey    TaskKey
	TaskStatus TaskStatus
	Context    []byte
	Argument   []byte
	Extra      TaskExtra
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s TaskModel) updateMap() map[string]interface{} { // TODO
	return map[string]interface{}{
		"task_status": s.TaskStatus,
		"context":     s.Context,
		"argument":    s.Argument,
		"extra":       s.Extra,
	}
}

type TaskExtra struct {
	// ErrStrs []string TODO: need store errStrs
}

func (s TaskExtra) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *TaskExtra) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	return json.Unmarshal(v.([]byte), s)
}
