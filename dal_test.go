package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func Test_taskDALImp_GetInitialized(t *testing.T) {
	convey.Convey("Test_taskDALImp_GetInitialized", t, func() {
		db := testDB("Test_taskDALImp_GetInitialized")
		tdal := taskDALImp{config: &TaskConfig{DB: db, Table: "tasks"}}
		convey.Convey("only has sensitive keys", func() {
			convey.Convey("normal", func() {
				_ = tdal.Create(db, &Task{TaskKey: "t1", TaskStatus: TaskStatusInitialized, CreatedAt: time.Now(), UpdatedAt: time.Now()})
				task, err := tdal.GetInitialized(db, []TaskKey{"t1"}, time.Second, nil)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
			})
			convey.Convey("invalid time", func() {
				_ = tdal.Create(db, &Task{TaskKey: "t1", TaskStatus: TaskStatusInitialized, CreatedAt: time.Now(), UpdatedAt: time.Now()})
				task, err := tdal.GetInitialized(db, []TaskKey{"t1"}, -time.Second, nil)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldBeNil)
			})
		})
	})
}
