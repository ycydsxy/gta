package gta

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func Test_taskSchedulerImp_CreateTask(t *testing.T) {
	convey.Convey("Test_taskSchedulerImp_CreateTask", t, func() {
		tc, _ := newConfig(testDB("Test_taskSchedulerImp_CreateTask"), "tasks")
		tr := tc.taskRegister
		tdal := &taskDALImp{config: tc}
		tass := &taskAssemblerImp{config: tc}
		pool, _ := ants.NewPool(tc.PoolSize, ants.WithLogger(tc.logger()), ants.WithNonblocking(true))
		tsch := &taskSchedulerImp{config: tc, register: tr, dal: tdal, assembler: tass, pool: pool}
		_ = tr.Register("t1", TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }})
		_ = tr.Register("t2", TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return ErrUnexpected }})

		convey.Convey("normal", func() {
			convey.Convey("built in transaction", func() {
				convey.Convey("transaction succeeded", func() {
					convey.Convey("not full pool", func() {
						db := tc.DB.Set(transactionKey, &sync.Map{})
						err := db.Transaction(func(tx *gorm.DB) error {
							if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
								return err
							}
							if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
								return err
							}
							return nil
						})
						convey.So(err, convey.ShouldBeNil)
						convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
						m, _ := db.Get(transactionKey)
						t1, ok1 := m.(*sync.Map).Load(uint64(1))
						convey.So(ok1, convey.ShouldBeTrue)
						convey.So(t1.(*Task).TaskKey, convey.ShouldEqual, "t1")
						t2, ok2 := m.(*sync.Map).Load(uint64(2))
						convey.So(ok2, convey.ShouldBeTrue)
						convey.So(t2.(*Task).TaskKey, convey.ShouldEqual, "t2")
						task1, _ := tdal.Get(tc.DB, 1)
						convey.So(task1.TaskKey, convey.ShouldEqual, "t1")
						convey.So(task1.TaskStatus, convey.ShouldEqual, TaskStatusRunning)
						task2, _ := tdal.Get(tc.DB, 2)
						convey.So(task2.TaskKey, convey.ShouldEqual, "t2")
						convey.So(task2.TaskStatus, convey.ShouldEqual, TaskStatusRunning)
					})
					convey.Convey("full pool", func() {
						pool, _ := ants.NewPool(1, ants.WithLogger(tc.logger()), ants.WithNonblocking(true))
						_ = pool.Submit(func() { time.Sleep(time.Second * 10) })
						tsch.pool = pool
						db := tc.DB.Set(transactionKey, &sync.Map{})
						err := db.Transaction(func(tx *gorm.DB) error {
							if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
								return err
							}
							if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
								return err
							}
							return nil
						})
						convey.So(err, convey.ShouldBeNil)
						convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
						m, _ := db.Get(transactionKey)
						_, ok1 := m.(*sync.Map).Load(uint64(1))
						convey.So(ok1, convey.ShouldBeFalse)
						_, ok2 := m.(*sync.Map).Load(uint64(2))
						convey.So(ok2, convey.ShouldBeFalse)
						task1, _ := tdal.Get(tc.DB, 1)
						convey.So(task1.TaskKey, convey.ShouldEqual, "t1")
						convey.So(task1.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
						task2, _ := tdal.Get(tc.DB, 2)
						convey.So(task2.TaskKey, convey.ShouldEqual, "t2")
						convey.So(task2.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
					})
				})
				convey.Convey("transaction failed", func() {
					db := tc.DB.Set(transactionKey, &sync.Map{})
					err := db.Transaction(func(tx *gorm.DB) error {
						if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
							return err
						}
						if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
							return err
						}
						return ErrUnexpected
					})
					convey.So(err, convey.ShouldNotBeNil)
					convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
					task1, _ := tdal.Get(tc.DB, 1)
					convey.So(task1, convey.ShouldBeNil)
					task2, _ := tdal.Get(tc.DB, 2)
					convey.So(task2, convey.ShouldBeNil)
				})
			})
			convey.Convey("non built in transaction", func() {
				convey.Convey("transaction succeeded", func() {
					db := tc.DB
					err := db.Transaction(func(tx *gorm.DB) error {
						if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
							return err
						}
						if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
							return err
						}
						return nil
					})
					convey.So(err, convey.ShouldBeNil)
					convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
					_, ok := db.Get(transactionKey)
					convey.So(ok, convey.ShouldBeFalse)
					task1, _ := tdal.Get(tc.DB, 1)
					convey.So(task1.TaskKey, convey.ShouldEqual, "t1")
					convey.So(task1.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
					task2, _ := tdal.Get(tc.DB, 2)
					convey.So(task2.TaskKey, convey.ShouldEqual, "t2")
					convey.So(task2.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
				})
				convey.Convey("transaction failed", func() {
					db := tc.DB
					err := db.Transaction(func(tx *gorm.DB) error {
						if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
							return err
						}
						if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
							return err
						}
						return ErrUnexpected
					})
					_, ok := db.Get(transactionKey)
					convey.So(ok, convey.ShouldBeFalse)
					convey.So(err, convey.ShouldNotBeNil)
					convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
					task1, _ := tdal.Get(tc.DB, 1)
					convey.So(task1, convey.ShouldBeNil)
					task2, _ := tdal.Get(tc.DB, 2)
					convey.So(task2, convey.ShouldBeNil)
				})
			})
		})

		convey.Convey("ctx cancelled", func() {
			tc.cancel()

			err := tsch.CreateTask(tc.DB, context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)
			convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
			task1, _ := tdal.Get(tc.DB, 1)
			convey.So(task1.TaskKey, convey.ShouldEqual, "t1")
			convey.So(task1.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
		})

		convey.Convey("dry run mode", func() {
			tc.DryRun = true

			convey.Convey("built in transaction", func() {
				db := tc.DB.Set(transactionKey, &sync.Map{})
				err := db.Transaction(func(tx *gorm.DB) error {
					if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
						return err
					}
					if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
				m, _ := db.Get(transactionKey)
				var count int
				m.(*sync.Map).Range(func(key, value interface{}) bool {
					convey.So(value.(*Task).TaskKey, convey.ShouldBeIn, []TaskKey{"t1", "t2"})
					count++
					return true
				})
				convey.So(count, convey.ShouldEqual, 2)
				task1, _ := tdal.Get(tc.DB, 1)
				convey.So(task1, convey.ShouldBeNil)
				task2, _ := tdal.Get(tc.DB, 2)
				convey.So(task2, convey.ShouldBeNil)
			})
			convey.Convey("non built in transaction", func() {
				db := tc.DB
				err := db.Transaction(func(tx *gorm.DB) error {
					if err := tsch.CreateTask(tx, context.TODO(), "t1", nil); err != nil {
						return err
					}
					if err := tsch.CreateTask(tx, context.TODO(), "t2", nil); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(tsch.runningTaskIDs(), convey.ShouldHaveLength, 0)
				_, ok := db.Get(transactionKey)
				convey.So(ok, convey.ShouldBeFalse)
				task1, _ := tdal.Get(tc.DB, 1)
				convey.So(task1, convey.ShouldBeNil)
				task2, _ := tdal.Get(tc.DB, 2)
				convey.So(task2, convey.ShouldBeNil)
			})
		})

		convey.Convey("error", func() {

		})
	})
}

func Test_taskSchedulerImp_GoScheduleTask(t *testing.T) {
	convey.Convey("Test_taskSchedulerImp_GoScheduleTask", t, func() {
		tc, _ := newConfig(testDB("Test_taskSchedulerImp_GoScheduleTask"), "tasks", WithPoolSize(1))
		tr := tc.taskRegister
		tdal := &taskDALImp{config: tc}
		tass := &taskAssemblerImp{config: tc}
		pool, _ := ants.NewPool(tc.PoolSize, ants.WithLogger(tc.logger()), ants.WithNonblocking(true))
		tsch := &taskSchedulerImp{config: tc, register: tr, dal: tdal, assembler: tass, pool: pool}
		var t1Run int64
		_ = tr.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})

		convey.Convey("wrong status", func() {
			tsch.GoScheduleTask(&Task{ID: 10001, TaskKey: "t1"})
			time.Sleep(time.Second)
			convey.So(t1Run, convey.ShouldEqual, 0)
		})

		convey.Convey("full pool", func() {
			_ = pool.Submit(func() { time.Sleep(time.Hour) })
			tsch.GoScheduleTask(&Task{ID: 10001, TaskKey: "t1", TaskStatus: TaskStatusRunning})
			time.Sleep(time.Second)
			convey.So(t1Run, convey.ShouldEqual, 1)
		})

		convey.Convey("error", func() {
			pool.Release()
			tsch.GoScheduleTask(&Task{ID: 10001, TaskKey: "t1", TaskStatus: TaskStatusRunning})
			time.Sleep(time.Second)
			convey.So(t1Run, convey.ShouldEqual, 0)
		})
	})
}
