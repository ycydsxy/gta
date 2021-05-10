package gta

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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

func TestTaskManager_Run(t *testing.T) {
	countHandler := func(run *int64) func(ctx context.Context, arg interface{}) (err error) {
		return func(ctx context.Context, arg interface{}) (err error) {
			atomic.AddInt64(run, 1)
			return nil
		}
	}
	convey.Convey("TestTaskManager_Run", t, func() {
		convey.Convey("normal", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
			m.Start()
			err1 := m.Run(context.TODO(), "t1", nil)
			err2 := m.Run(context.TODO(), "t1", nil)
			m.Stop(true)
			convey.So(err1, convey.ShouldBeNil)
			convey.So(err2, convey.ShouldBeNil)
			convey.So(t1Run, convey.ShouldEqual, 2)
		})

		convey.Convey("run after cancel", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks")
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
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
			m.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
			err := m.Run(context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)
			time.Sleep(time.Second)
			convey.So(t1Run, convey.ShouldEqual, 1)
		})

		convey.Convey("no enough pool", func() {
			m := NewTaskManager(testDB("TestTaskManager_Run"), "tasks", WithPoolSize(5))
			var t1Run int64
			m.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
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
			m.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
			m.Start()
			m.Stop(true)
			err := m.Run(context.TODO(), "t1", nil)
			convey.So(err, convey.ShouldBeNil)

			m2 := NewTaskManager(defaultDB, defaultTable)
			m2.Register("t1", TaskDefinition{Handler: countHandler(&t1Run)})
			m2.Start()
			time.Sleep(time.Second)
			m.Stop(true)
			convey.So(t1Run, convey.ShouldEqual, 1)
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
}

func TestTaskManager_Transaction(t *testing.T) {
}

func TestTaskManager_ForceRerunTask(t *testing.T) {
}

func TestTaskManager_ForceRerunTasks(t *testing.T) {
}

func TestTaskManager_QueryUnsuccessfulTasks(t *testing.T) {
}

func TestTaskManager_Stop(t *testing.T) {
}
