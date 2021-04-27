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

/*  ---------------------------- TABLE SCHEMA ----------------------------
CREATE TABLE `async_task_test` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `task_key` varchar(64) NOT NULL DEFAULT '',
  `task_status` varchar(64) NOT NULL DEFAULT '',
  `context` mediumtext,
  `argument` mediumtext,
  `extra` mediumtext,
  `created_at` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  `updated_at` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  KEY `idx_task_key` (`task_key`),
  KEY `idx_task_status` (`task_status`),
  KEY `idx_updated_at` (`updated_at`)
) ENGINE=InnoDB AUTO_INCREMENT=10000 DEFAULT CHARSET=utf8mb4;
*/

type TaskModel struct {
	ID         uint64
	TaskKey    TaskKey
	TaskStatus TaskStatus
	Context    json.RawMessage
	Argument   json.RawMessage
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
	return valueJSON(s)
}

func (s *TaskExtra) Scan(v interface{}) error {
	return scanJSON(v, s)
}
