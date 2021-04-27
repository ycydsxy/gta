package gta

import (
	"time"

	"github.com/jinzhu/gorm"
)

type taskDAL interface {
	GetInitializedTask(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration, insensitiveKeys []TaskKey) (TaskModel, error)
	UpdateTaskStatus(tx *gorm.DB, task TaskModel, newStatus TaskStatus) (int64, error)
	UpdateTaskStatusByIDs(tx *gorm.DB, taskIDs []uint64, oldStatus TaskStatus, newStatus TaskStatus) (int64, error)
	CreateTask(tx *gorm.DB, task *TaskModel) error
	GetSliceByOffsetsAndTaskStatus(tx *gorm.DB, startOffset, endOffset time.Duration, taskStatus TaskStatus, ) ([]TaskModel, error)
	GetForUpdate(tx *gorm.DB, id uint64) (TaskModel, error)
	HardDeleteSucceededTaskByOffset(tx *gorm.DB, offset time.Duration, excludeTaskKeys []TaskKey) (int64, error)
	HardDeleteTaskByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error)
	UpdateByIDAndKey(tx *gorm.DB, id uint64, key TaskKey, updateMap map[string]interface{}) (int64, error)
	GetSliceExcludeSucceeded(tx *gorm.DB, excludeKeys []TaskKey) ([]TaskModel, error)
	ForceUpdateTaskStatusByIDs(tx *gorm.DB, ids []uint64, oriStatus TaskStatus, newStatus TaskStatus) (int64, error)
}

type taskDALImp struct {
	config *Config
}

func (s *taskDALImp) GetInitializedTask(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration,
	insensitiveKeys []TaskKey) (TaskModel, error) {
	var rule TaskModel

	db := tx.Table(s.config.TableName).Where("task_status = ?", taskStatusInitialized)
	if len(sensitiveKeys) > 0 && len(insensitiveKeys) > 0 {
		db = db.Where("(updated_at >= ? AND task_key IN (?)) OR (task_key IN (?))", time.Now().Add(-offset),
			sensitiveKeys, insensitiveKeys)
	} else if len(sensitiveKeys) > 0 {
		db = db.Where("updated_at >= ? AND task_key IN (?)", time.Now().Add(-offset), sensitiveKeys)
	} else if len(insensitiveKeys) > 0 {
		db = db.Where("task_key IN (?)", insensitiveKeys)
	} else {
		return rule, ErrUnexpected
	}

	err := db.First(&rule).Error
	return rule, err
}

func (s *taskDALImp) UpdateTaskStatus(tx *gorm.DB, task TaskModel, newStatus TaskStatus) (int64, error) {
	db := tx.Table(s.config.TableName).Where("id = ? AND task_status = ?",
		task.ID, task.TaskStatus).Update("task_status", newStatus)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) UpdateTaskStatusByIDs(tx *gorm.DB, taskIDs []uint64,
	oldStatus TaskStatus, newStatus TaskStatus) (int64, error) {
	db := tx.Table(s.config.TableName).Where("id IN (?) AND task_status = ?",
		taskIDs, oldStatus).Update("task_status", newStatus)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) CreateTask(tx *gorm.DB, task *TaskModel) error {
	return tx.Table(s.config.TableName).Create(&task).Error
}

func (s *taskDALImp) GetSliceByOffsetsAndTaskStatus(tx *gorm.DB, startOffset, endOffset time.Duration,
	taskStatus TaskStatus, ) ([]TaskModel, error) {
	timeNow := time.Now()
	var res []TaskModel
	err := tx.Table(s.config.TableName).Where("task_status = ? AND updated_at BETWEEN ? AND ?",
		taskStatus, timeNow.Add(-startOffset), timeNow.Add(-endOffset)).Find(&res).Error
	return res, err
}

func (s *taskDALImp) GetForUpdate(tx *gorm.DB, id uint64) (TaskModel, error) {
	var rule TaskModel
	err := tx.Table(s.config.TableName).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&rule).Error
	return rule, err
}

func (s *taskDALImp) HardDeleteSucceededTaskByOffset(tx *gorm.DB, offset time.Duration,
	excludeTaskKeys []TaskKey) (int64, error) {
	var rule TaskModel
	db := tx.Table(s.config.TableName).Where("task_status = ? AND updated_at < ? AND task_key NOT IN (?)",
		taskStatusSucceeded, time.Now().Add(-offset), excludeTaskKeys).Delete(&rule)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) HardDeleteTaskByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error) {
	var rule TaskModel
	db := tx.Table(s.config.TableName).Where("task_status = ? AND id = ?", status, id).Delete(&rule)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) UpdateByIDAndKey(tx *gorm.DB, id uint64, key TaskKey, updateMap map[string]interface{}) (int64,
	error) {
	db := tx.Table(s.config.TableName).Where("id = ? AND task_key = ?", id, key).Updates(updateMap)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) GetSliceExcludeSucceeded(tx *gorm.DB, excludeKeys []TaskKey) ([]TaskModel, error) {
	var res []TaskModel
	err := tx.Table(s.config.TableName).Where("task_status <> ? AND task_key NOT IN (?)",
		taskStatusSucceeded, excludeKeys).Find(&res).Error
	return res, err
}

func (s *taskDALImp) ForceUpdateTaskStatusByIDs(tx *gorm.DB, ids []uint64, oriStatus TaskStatus,
	newStatus TaskStatus) (int64, error) {
	db := tx.Table(s.config.TableName).Where("id IN (?) AND task_status = ?",
		ids, oriStatus).Updates(map[string]interface{}{
		"task_status": newStatus,
		"updated_at":  time.Now(),
	})
	return db.RowsAffected, db.Error
}
