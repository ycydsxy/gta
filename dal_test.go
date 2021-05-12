package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func Test_taskDALImp_GetInitialized(t *testing.T) {
	convey.Convey("Test_taskDALImp_GetInitialized", t, func() {
		db := testDB("Test_taskDALImp_GetInitialized")
		convey.Convey("normal", func() {
			tdal := taskDALImp{config: &TaskConfig{DB: db, Table: "tasks"}}
			convey.Convey("only has sensitive keys", func() {
				convey.Convey("normal time", func() {
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
		convey.Convey("error", func() {
			tdal := taskDALImp{config: &TaskConfig{DB: db, Table: "not exist"}}
			_, err := tdal.GetInitialized(db, nil, time.Second, nil)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func Test_taskDALImp_Get(t *testing.T) {
	convey.Convey("Test_taskDALImp_Get", t, func() {
		convey.Convey("error", func() {
			db := testDB("Test_taskDALImp_Get")
			tdal := taskDALImp{config: &TaskConfig{DB: db, Table: "not exist"}}
			_, err := tdal.Get(db, 1)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func Test_taskDALImp_GetForUpdate(t *testing.T) {
	convey.Convey("Test_taskDALImp_GetForUpdate", t, func() {
		convey.Convey("error", func() {
			db := testDB("Test_taskDALImp_GetForUpdate")
			tdal := taskDALImp{config: &TaskConfig{DB: db, Table: "not exist"}}
			_, err := tdal.GetForUpdate(db, 1)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
