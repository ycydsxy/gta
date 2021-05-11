package gta

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	randomIntervalFactor = 0.2
)

// randomInterval generates random interval in [interval,randomIntervalFactor*interval)
func randomInterval(interval time.Duration) time.Duration {
	return interval + time.Duration(randomIntervalFactor*rand.Float64()*float64(interval))
}

func panicHandler() {
	if r := recover(); r != nil {
		logrus.Errorf("panic: %v\n%s", r, string(debug.Stack()))
	}
}

func minInt64(i ...int64) int64 {
	min := int64(math.MaxInt64)
	for _, a := range i {
		if a < min {
			min = a
		}
	}
	return min
}

func testDB(dbName string) *gorm.DB {
	dbName = dbName + fmt.Sprintf("_%d.db", rand.Int())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic(err)
	}
	if err = db.Migrator().AutoMigrate(&Task{}); err != nil {
		panic(err)
	}
	return db
}

func testCountHandler(run *int64) TaskHandler {
	return func(ctx context.Context, arg interface{}) (err error) {
		atomic.AddInt64(run, 1)
		return nil
	}
}

func testWrappedHandler(fs ...TaskHandler) TaskHandler {
	return func(ctx context.Context, arg interface{}) (err error) {
		for _, f := range fs {
			if err := f(ctx, arg); err != nil {
				return err
			}
		}
		return nil
	}
}
