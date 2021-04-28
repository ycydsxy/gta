package gta

import (
	"time"

	"github.com/jinzhu/gorm"
)

type taskDAL interface {
	GetInitialized(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration, insensitiveKeys []TaskKey) (*TaskModel, error)
	UpdateStatus(tx *gorm.DB, task TaskModel, newStatus TaskStatus) (int64, error)
	UpdateStatusByIDs(tx *gorm.DB, taskIDs []uint64, ori TaskStatus, new TaskStatus) (int64, error)
	Create(tx *gorm.DB, task *TaskModel) error
	GetSliceByOffsetsAndStatus(tx *gorm.DB, startOffset, endOffset time.Duration, status TaskStatus) ([]TaskModel, error)
	GetForUpdate(tx *gorm.DB, id uint64) (*TaskModel, error)
	HardDeleteSucceededByOffset(tx *gorm.DB, offset time.Duration, excludeKeys []TaskKey) (int64, error)
	HardDeleteByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error)
	UpdateByIDAndKey(tx *gorm.DB, id uint64, key TaskKey, updateMap map[string]interface{}) (int64, error)
	GetSliceExcludeSucceeded(tx *gorm.DB, excludeKeys []TaskKey) ([]TaskModel, error)
}

type taskDALImp struct {
	config *Config
}

func (s *taskDALImp) GetInitialized(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration,
	insensitiveKeys []TaskKey) (*TaskModel, error) {
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
		return nil, ErrUnexpected
	}

	if err := db.First(&rule).Error; err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &rule, nil
}

func (s *taskDALImp) UpdateStatus(tx *gorm.DB, task TaskModel, newStatus TaskStatus) (int64, error) {
	db := tx.Table(s.config.TableName).Where("id = ? AND task_status = ?",
		task.ID, task.TaskStatus).Update("task_status", newStatus)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) UpdateStatusByIDs(tx *gorm.DB, ids []uint64, ori TaskStatus, new TaskStatus) (int64, error) {
	db := tx.Table(s.config.TableName).Where("id IN (?) AND task_status = ?", ids, ori).
		Updates(map[string]interface{}{"task_status": new, "updated_at": time.Now()}) // TODO
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) Create(tx *gorm.DB, task *TaskModel) error {
	return tx.Table(s.config.TableName).Create(&task).Error
}

func (s *taskDALImp) GetSliceByOffsetsAndStatus(tx *gorm.DB, startOffset, endOffset time.Duration,
	status TaskStatus) ([]TaskModel, error) {
	timeNow := time.Now()
	var res []TaskModel
	err := tx.Table(s.config.TableName).Where("task_status = ? AND updated_at BETWEEN ? AND ?",
		status, timeNow.Add(-startOffset), timeNow.Add(-endOffset)).Find(&res).Error
	return res, err
}

func (s *taskDALImp) GetForUpdate(tx *gorm.DB, id uint64) (*TaskModel, error) {
	var rule TaskModel
	if err := tx.Table(s.config.TableName).Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", id).First(&rule).Error; err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *taskDALImp) HardDeleteSucceededByOffset(tx *gorm.DB, offset time.Duration, excludeKeys []TaskKey) (int64,
	error) {
	var rule TaskModel
	db := tx.Table(s.config.TableName).Where("task_status = ? AND updated_at < ? AND task_key NOT IN (?)",
		taskStatusSucceeded, time.Now().Add(-offset), excludeKeys).Delete(&rule)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) HardDeleteByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error) {
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
