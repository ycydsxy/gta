package gta

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// here are constants for task status
const (
	TaskStatusUnKnown     TaskStatus = ""
	TaskStatusInitialized TaskStatus = "initialized"
	TaskStatusRunning     TaskStatus = "running"
	TaskStatusSucceeded   TaskStatus = "succeeded"
	TaskStatusFailed      TaskStatus = "failed"
)

// TaskKey is a unique ID for a set of tasks with same definition.
type TaskKey string

// TaskStatus represents the status of a task.
type TaskStatus string

// Task is an entity in database.
type Task struct {
	ID         uint64
	TaskKey    TaskKey
	TaskStatus TaskStatus
	Context    []byte
	Argument   []byte
	Extra      TaskExtra
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskExtra contains other information of a task.
type TaskExtra struct{}

// Value implements Valuer.
func (s TaskExtra) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan implements Scanner.
func (s *TaskExtra) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	return json.Unmarshal(v.([]byte), s)
}
