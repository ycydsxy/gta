package gta

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func TestNewTaskManager(t *testing.T) {
	convey.Convey("TestNewTaskManager", t, func() {
		convey.Convey("normal", func() {
			m := NewTaskManager(&gorm.DB{}, "tasks")
			convey.So(m, convey.ShouldNotBeNil)
		})

		convey.Convey("error", func() {
			convey.So(func() { _ = NewTaskManager(&gorm.DB{}, "tasks", WithRunningTimeout(time.Hour*365)) }, convey.ShouldPanic)
		})
	})
}

func TestTaskManager_Start(t *testing.T) {
	convey.Convey("TestTaskManager_Start", t, func() {
		convey.Convey("normal", func() {
			m := NewTaskManager(testDB("TestTaskManager_Start"), "tasks")
			m.Start()
			defer m.Stop(false)
			convey.So(m.tr.GetBuiltInKeys(), convey.ShouldHaveLength, 2)
			task, err := m.tdal.Get(m.tc.DB, taskCheckAbnormalID)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
		})

		convey.Convey("dry run", func() {
			m := NewTaskManager(testDB("TestTaskManager_Start"), "tasks", WithDryRun(true))
			m.Start()
			defer m.Stop(false)
			convey.So(m.tr.GetBuiltInKeys(), convey.ShouldHaveLength, 0)
			task, err := m.tdal.Get(m.tc.DB, taskCheckAbnormalID)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldBeNil)
		})

	})
}

func TestTaskManager_Register(t *testing.T) {
	m := NewTaskManager(testDB("TestTaskManager_Register"), "tasks")
	convey.Convey("TestTaskManager_Register", t, func() {
		convey.Convey("normal", func() {
			m.Register("key1", TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }})
			def, err := m.tr.GetDefinition("key1")
			convey.So(err, convey.ShouldBeNil)
			convey.So(def, convey.ShouldNotBeNil)
		})

		convey.Convey("error", func() {
			convey.So(func() { m.Register("key1", TaskDefinition{}) }, convey.ShouldPanic)
		})
	})
}

type testGinCtxMarshaler struct{}

func (t *testGinCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	requestID := ctx.Value("request_id").(string)
	return json.Marshal(requestID)
}

func (t *testGinCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	var requestID string
	if err := json.Unmarshal(bytes, &requestID); err != nil {
		return nil, err
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("request_id", requestID)
	return ctx, nil
}

func TestTaskManager_Run(t *testing.T) {
	convey.Convey("TestTaskManager_Run", t, func() {
		convey.Convey("normal", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m.Start()
			err := m.Run(context.TODO(), "t1", nil)
			m.Stop(true)
			convey.So(err, convey.ShouldBeNil)
			convey.So(t1Run, convey.ShouldEqual, 1)
			task, err := m.tdal.Get(m.tc.DB, 10001)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
			convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusSucceeded)
		})

		convey.Convey("with options", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			convey.Convey("with CtxMarshaler", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{
					Handler: testWrappedHandler(func(ctx context.Context, arg interface{}) (err error) {
						_ = ctx.(*gin.Context)
						_ = ctx.Value("request_id").(string)
						return nil
					}, testCountHandler(&t1Run)),
					CtxMarshaler: &testGinCtxMarshaler{},
				})
				m.Start()
				err := m.Run(context.WithValue(context.TODO(), "request_id", "10086"), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
			})

			convey.Convey("with RetryTimes", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{
					Handler: testWrappedHandler(testCountHandler(&t1Run), func(ctx context.Context, arg interface{}) (err error) {
						return ErrUnexpected
					}),
					RetryTimes: 3,
				})
				m.Start()
				err := m.Run(context.TODO(), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 4)
			})

			convey.Convey("with RetryInterval", func() {
				var t1Run int64
				var timeSlice []time.Time
				m.Register("t1", TaskDefinition{
					Handler: testWrappedHandler(
						testCountHandler(&t1Run),
						func(ctx context.Context, arg interface{}) (err error) {
							timeSlice = append(timeSlice, time.Now())
							return nil
						},
						func(ctx context.Context, arg interface{}) (err error) {
							return ErrUnexpected
						},
					),
					RetryTimes:    1,
					RetryInterval: func(times int) time.Duration { return time.Millisecond * 100 },
				})
				m.Start()
				err := m.Run(context.TODO(), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 2)
				convey.So(timeSlice, convey.ShouldHaveLength, 2)
				sub := timeSlice[1].Sub(timeSlice[0])
				convey.So(sub, convey.ShouldBeGreaterThan, time.Millisecond*100)
				convey.So(sub, convey.ShouldBeLessThan, time.Second)
			})

			convey.Convey("with CleanSucceeded", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run), CleanSucceeded: true})
				m.Start()
				err := m.Run(context.TODO(), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
				task, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldBeNil)
			})

			convey.Convey("with InitTimeoutSensitive", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run), InitTimeoutSensitive: true})
				err1 := m.tdal.Create(m.tc.DB, &Task{
					ID:         10001,
					TaskKey:    "t1",
					TaskStatus: TaskStatusInitialized,
					Context:    nil,
					Argument:   nil,
					Extra:      TaskExtra{},
					CreatedAt:  time.Now().Add(-time.Hour),
					UpdatedAt:  time.Now().Add(-time.Hour),
				})
				err2 := m.tdal.Create(m.tc.DB, &Task{
					ID:         10002,
					TaskKey:    "t1",
					TaskStatus: TaskStatusInitialized,
					Context:    nil,
					Argument:   nil,
					Extra:      TaskExtra{},
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
				convey.So(err1, convey.ShouldBeNil)
				convey.So(err2, convey.ShouldBeNil)
				m.Start()
				time.Sleep(time.Second)
				m.Stop(true)
				convey.So(t1Run, convey.ShouldEqual, 1)
				task, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
				convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
			})
		})

		convey.Convey("ctx cancelled", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m.Start()
			m.Stop(false)
			err := m.Run(context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)
			task, err := m.tdal.Get(m.tc.DB, 10001)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
			convey.So(task.TaskKey, convey.ShouldEqual, "t1")
			convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
		})

		convey.Convey("dry run", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks", WithDryRun(true))
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			err := m.Run(context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)
			time.Sleep(time.Second)
			convey.So(t1Run, convey.ShouldEqual, 1)
		})

		convey.Convey("full pool", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks", WithPoolSize(5))
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m.Start()
			var errSlice []error
			for i := 0; i < 10; i++ {
				err := m.Run(context.TODO(), "t1", nil)
				if err != nil {
					errSlice = append(errSlice, err)
				}
			}
			m.Stop(true)
			convey.So(errSlice, convey.ShouldHaveLength, 0)
			convey.So(t1Run, convey.ShouldBeLessThan, 10)
			task, err := m.tdal.Get(m.tc.DB, 10010)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
			convey.So(task.TaskKey, convey.ShouldEqual, "t1")
			convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
		})

		convey.Convey("scan run", func() {
			defaultDB := testDB("TestTaskManager_Run")
			defaultTable := "tasks"
			m := NewTaskManager(defaultDB, defaultTable)
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m.Start()
			m.Stop(true)
			err := m.Run(context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)

			m2 := NewTaskManager(defaultDB, defaultTable)
			m2.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m2.Start()
			time.Sleep(time.Second)
			m.Stop(true)
			convey.So(t1Run, convey.ShouldEqual, 1)
		})

		convey.Convey("task failed", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			convey.Convey("task return error", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{Handler: testWrappedHandler(testCountHandler(&t1Run), func(ctx context.Context, arg interface{}) (err error) {
					return ErrUnexpected
				})})
				m.Start()
				err := m.Run(context.TODO(), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
				task, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
				convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusFailed)
			})

			convey.Convey("task panic inside", func() {
				var t1Run int64
				m.Register("t1", TaskDefinition{Handler: testWrappedHandler(testCountHandler(&t1Run), func(ctx context.Context, arg interface{}) (err error) {
					panic("panic inside task handler")
					return nil
				})})
				m.Start()
				err := m.Run(context.TODO(), "t1", nil)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
				task, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
				convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusFailed)
			})
		})

		convey.Convey("error", func() {
			convey.Convey("task not registered", func() {
				m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks", WithPoolSize(1))
				err := m.Run(context.TODO(), "not existed", nil)
				convey.So(err, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestTaskManager_RunWithTx(t *testing.T) {
	convey.Convey("TestTaskManager_RunWithTx", t, func() {
		m := NewTaskManager(testDB("TestTaskManager_RunWithTx"), "tasks", WithScanInterval(time.Second))
		var t1Run, t2Run int64
		m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
		m.Register("t2", TaskDefinition{Handler: testCountHandler(&t2Run)})

		convey.Convey("builtin transaction", func() {
			convey.Convey("transaction succeeded", func() {
				m.Start()
				err := m.Transaction(func(tx *gorm.DB) error {
					err1 := m.RunWithTx(tx, context.TODO(), "t1", nil)
					if err1 != nil {
						return err1
					}
					err2 := m.RunWithTx(tx, context.TODO(), "t2", nil)
					if err2 != nil {
						return err2
					}
					return nil
				})
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
				convey.So(t2Run, convey.ShouldEqual, 1)
			})

			convey.Convey("transaction failed", func() {
				m.Start()
				err := m.Transaction(func(tx *gorm.DB) error {
					err1 := m.RunWithTx(tx, context.TODO(), "t1", nil)
					if err1 != nil {
						return err1
					}
					return ErrUnexpected
					// cannot reach here
					err2 := m.RunWithTx(tx, context.TODO(), "t2", nil)
					if err2 != nil {
						return err2
					}
					return nil
				})
				m.Stop(true)
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(t1Run, convey.ShouldEqual, 0)
				convey.So(t2Run, convey.ShouldEqual, 0)
				task1, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task1, convey.ShouldBeNil)
				task2, err := m.tdal.Get(m.tc.DB, 10002)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task2, convey.ShouldBeNil)
			})
		})

		convey.Convey("not builtin transaction", func() {
			convey.Convey("transaction succeeded", func() {
				m.Start()
				err := m.tc.DB.Transaction(func(tx *gorm.DB) error {
					err1 := m.RunWithTx(tx, context.TODO(), "t1", nil)
					if err1 != nil {
						return err1
					}
					err2 := m.RunWithTx(tx, context.TODO(), "t2", nil)
					if err2 != nil {
						return err2
					}
					return nil
				})
				time.Sleep(time.Second)
				m.Stop(true)
				convey.So(err, convey.ShouldBeNil)
				convey.So(t1Run, convey.ShouldEqual, 1)
				convey.So(t2Run, convey.ShouldEqual, 1)
			})
			convey.Convey("transaction failed", func() {
				m.Start()
				err := m.tc.DB.Transaction(func(tx *gorm.DB) error {
					err1 := m.RunWithTx(tx, context.TODO(), "t1", nil)
					if err1 != nil {
						return err1
					}
					return ErrUnexpected
					// cannot reach here
					err2 := m.RunWithTx(tx, context.TODO(), "t2", nil)
					if err2 != nil {
						return err2
					}
					return nil
				})
				m.Stop(true)
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(t1Run, convey.ShouldEqual, 0)
				convey.So(t2Run, convey.ShouldEqual, 0)
				task1, err := m.tdal.Get(m.tc.DB, 10001)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task1, convey.ShouldBeNil)
				task2, err := m.tdal.Get(m.tc.DB, 10002)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task2, convey.ShouldBeNil)
			})
		})
	})
}

func TestTaskManager_Transaction(t *testing.T) {
	convey.Convey("TestTaskManager_Transaction", t, func() {
		m := NewTaskManager(testDB("TestTaskManager_Transaction"), "tasks")
		convey.Convey("normal", func() {
			err := m.Transaction(func(tx *gorm.DB) error { return nil })
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("error", func() {
			err := m.Transaction(func(tx *gorm.DB) error { return ErrUnexpected })
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func TestTaskManager_Stop(t *testing.T) {
	convey.Convey("TestTaskManager_Stop", t, func() {
		m := NewTaskManager(testDB("TestTaskManager_Stop"), "tasks", WithWaitTimeout(time.Millisecond))
		convey.Convey("wait", func() {
			m.Register("t1", TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }})
			m.Start()
			err := m.Run(context.TODO(), "t1", nil)
			m.Stop(true)
			convey.So(err, convey.ShouldBeNil)
			task, err := m.tdal.Get(m.tc.DB, 10001)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
			convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusSucceeded)
		})

		convey.Convey("not wait", func() {
			m.Register("t1", TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) {
				time.Sleep(time.Second * 6)
				return nil
			}})
			m.Start()
			err := m.Run(context.TODO(), "t1", nil)
			m.Stop(false)
			convey.So(err, convey.ShouldBeNil)
			task, err := m.tdal.Get(m.tc.DB, 10001)
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldNotBeNil)
			convey.So(task.TaskStatus, convey.ShouldEqual, TaskStatusInitialized)
		})
	})
}

func TestTaskManager_ForceRerunTasks(t *testing.T) {
	m := NewTaskManager(testDB("TestTaskManager_ForceRerunTasks"), "tasks")
	var t1Run int64
	m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
	convey.Convey("TestTaskManager_ForceRerunTasks", t, func() {
		_ = m.tdal.Create(m.tc.DB, &Task{
			ID:         10001,
			TaskKey:    "t1",
			TaskStatus: TaskStatusFailed,
			Context:    nil,
			Argument:   nil,
			Extra:      TaskExtra{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
		count, err := m.ForceRerunTasks([]uint64{10001}, TaskStatusFailed)
		convey.So(err, convey.ShouldBeNil)
		convey.So(count, convey.ShouldEqual, 1)
	})
}

func TestTaskManager_QueryUnsuccessfulTasks(t *testing.T) {
	m := NewTaskManager(testDB("TestTaskManager_QueryUnsuccessfulTasks"), "tasks")
	var t1Run int64
	m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
	convey.Convey("TestTaskManager_QueryUnsuccessfulTasks", t, func() {
		_ = m.tdal.Create(m.tc.DB, &Task{
			ID:         10001,
			TaskKey:    "t1",
			TaskStatus: TaskStatusFailed,
			Context:    nil,
			Argument:   nil,
			Extra:      TaskExtra{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
		tasks, err := m.QueryUnsuccessfulTasks(10, 0)
		convey.So(err, convey.ShouldBeNil)
		convey.So(tasks, convey.ShouldHaveLength, 1)
	})
}

func TestTaskManager_Others(t *testing.T) {
	convey.Convey("TestTaskManager_Others", t, func() {
		m := NewTaskManager(testDB("TestTaskManager_Others"), "tasks")
		convey.Convey("nested tasks", func() {
			var innerRun, outterRun int64
			m.Register("inner", TaskDefinition{Handler: testCountHandler(&innerRun)})
			m.Register("outter", TaskDefinition{Handler: testWrappedHandler(testCountHandler(&outterRun), func(ctx context.Context, arg interface{}) (err error) {
				return m.Run(context.TODO(), "inner", nil)
			})})

			m.Start()
			err := m.Run(context.TODO(), "outter", nil)
			convey.So(err, convey.ShouldBeNil)
			time.Sleep(time.Second)
			m.Stop(true)
			convey.So(outterRun, convey.ShouldEqual, 1)
			convey.So(innerRun, convey.ShouldEqual, 1)
		})

		convey.Convey("multiple tasks", func() {
			var t1Run, t2Run int64
			m.Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			m.Register("t2", TaskDefinition{Handler: testCountHandler(&t2Run)})

			m.Start()
			err := m.Transaction(func(tx *gorm.DB) error {
				if err := m.RunWithTx(tx, context.TODO(), "t1", nil); err != nil {
					return err
				}
				if err := m.RunWithTx(tx, context.TODO(), "t2", nil); err != nil {
					return err
				}
				return nil
			})
			convey.So(err, convey.ShouldBeNil)
			m.Stop(true)
			convey.So(t1Run, convey.ShouldEqual, 1)
			convey.So(t2Run, convey.ShouldEqual, 1)
		})
	})
}
