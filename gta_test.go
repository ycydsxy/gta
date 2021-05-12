package gta

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestMainProcess(t *testing.T) {
	convey.Convey("TestMainProcess", t, func() {
		var t1Run, t2Run int64
		// register before start
		Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
		// start
		StartWithOptions(testDB("TestMainProcess"), "tasks")
		// register after start
		Register("t2", TaskDefinition{Handler: testCountHandler(&t2Run)})

		err1 := Run(context.TODO(), "t1", nil)
		err2 := Run(context.TODO(), "t2", nil)

		Stop(true)

		convey.So(err1, convey.ShouldBeNil)
		convey.So(err2, convey.ShouldBeNil)
		convey.So(t1Run, convey.ShouldEqual, 1)
		convey.So(t2Run, convey.ShouldEqual, 1)
	})
}

func TestDefaultManager(t *testing.T) {
	convey.Convey("TestDefaultManager", t, func() {
		m := DefaultManager()
		convey.So(m, convey.ShouldNotBeNil)
	})
}
