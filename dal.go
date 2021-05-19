package gta

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type taskDAL interface {
	Create(tx *gorm.DB, task *Task) error

	Get(tx *gorm.DB, id uint64) (*Task, error)
	GetForUpdate(tx *gorm.DB, id uint64) (*Task, error)
	GetInitialized(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration, insensitiveKeys []TaskKey) (*Task, error)
	GetSliceByOffsetsAndStatus(tx *gorm.DB, startOffset, endOffset time.Duration, status TaskStatus) ([]Task, error)
	GetSliceExcludeSucceeded(tx *gorm.DB, excludeKeys []TaskKey, limit, offset int) ([]Task, error)

	Update(tx *gorm.DB, task *Task) (int64, error)
	UpdateStatusByIDs(tx *gorm.DB, taskIDs []uint64, ori TaskStatus, new TaskStatus) (int64, error)

	DeleteSucceededByOffset(tx *gorm.DB, offset time.Duration, excludeKeys []TaskKey) (int64, error)
	DeleteByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error)
}

type taskDALImp struct {
	*options
}

func (s *taskDALImp) tabledDB(tx *gorm.DB) *gorm.DB {
	return tx.Table(s.table)
}

func (s *taskDALImp) Create(tx *gorm.DB, task *Task) error {
	return s.tabledDB(tx).Create(&task).Error
}

func (s *taskDALImp) Get(tx *gorm.DB, id uint64) (*Task, error) {
	var rule Task
	if err := s.tabledDB(tx).Where("id = ?", id).Take(&rule).Error; err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *taskDALImp) GetForUpdate(tx *gorm.DB, id uint64) (*Task, error) {
	var rule Task
	if err := s.tabledDB(tx).Where("id = ?", id).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&rule).
		Error; err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *taskDALImp) GetInitialized(tx *gorm.DB, sensitiveKeys []TaskKey, offset time.Duration,
	insensitiveKeys []TaskKey) (*Task, error) {
	var rule Task

	db := s.tabledDB(tx).Where("task_status = ?", TaskStatusInitialized)
	if len(sensitiveKeys) > 0 && len(insensitiveKeys) > 0 {
		db = db.Where("(updated_at >= ? AND task_key IN (?)) OR (task_key IN (?))", time.Now().Add(-offset),
			sensitiveKeys, insensitiveKeys)
	} else if len(sensitiveKeys) > 0 {
		db = db.Where("updated_at >= ? AND task_key IN (?)", time.Now().Add(-offset), sensitiveKeys)
	} else if len(insensitiveKeys) > 0 {
		db = db.Where("task_key IN (?)", insensitiveKeys)
	}

	if err := db.Take(&rule).Error; err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &rule, nil
}

func (s *taskDALImp) GetSliceByOffsetsAndStatus(tx *gorm.DB, startOffset, endOffset time.Duration,
	status TaskStatus) ([]Task, error) {
	timeNow := time.Now()
	var res []Task
	err := s.tabledDB(tx).Where("task_status = ? AND updated_at BETWEEN ? AND ?",
		status, timeNow.Add(-startOffset), timeNow.Add(-endOffset)).Find(&res).Error
	return res, err
}

func (s *taskDALImp) GetSliceExcludeSucceeded(tx *gorm.DB, excludeKeys []TaskKey, limit, offset int) ([]Task, error) {
	var res []Task
	db := s.tabledDB(tx).Where("task_status <> ?", TaskStatusSucceeded)
	if len(excludeKeys) > 0 {
		db = db.Where("task_key NOT IN (?)", excludeKeys)
	}
	err := db.Limit(limit).Offset(offset).Find(&res).Error
	return res, err
}

func (s *taskDALImp) Update(tx *gorm.DB, task *Task) (int64, error) {
	db := s.tabledDB(tx).Updates(task)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) UpdateStatusByIDs(tx *gorm.DB, ids []uint64, oriStatus TaskStatus, newStatus TaskStatus) (int64, error) {
	db := s.tabledDB(tx).Where("id IN (?) AND task_status = ?", ids, oriStatus).Updates(&Task{TaskStatus: newStatus})
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) DeleteSucceededByOffset(tx *gorm.DB, offset time.Duration, excludeKeys []TaskKey) (int64,
	error) {
	var rule Task
	db := s.tabledDB(tx).Where("task_status = ? AND updated_at < ?", TaskStatusSucceeded, time.Now().Add(-offset))
	if len(excludeKeys) > 0 {
		db = db.Where("task_key NOT IN (?)", excludeKeys)
	}
	db.Delete(&rule)
	return db.RowsAffected, db.Error
}

func (s *taskDALImp) DeleteByIDAndStatus(tx *gorm.DB, id uint64, status TaskStatus) (int64, error) {
	var rule Task
	db := s.tabledDB(tx).Where("task_status = ? AND id = ?", status, id).Delete(&rule)
	return db.RowsAffected, db.Error
}
