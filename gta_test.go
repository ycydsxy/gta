package gta

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func TestMainProcess(t *testing.T) {
	convey.Convey("TestMainProcess", t, func() {
		convey.Convey("not started", func() {
			convey.So(func() { _ = Run(context.TODO(), "t1", nil) }, convey.ShouldPanic)
			convey.So(func() { _ = RunWithTx(testDB("TestMainProcess"), context.TODO(), "t1", nil) }, convey.ShouldPanic)
			convey.So(func() { _ = Transaction(func(tx *gorm.DB) error { return nil }) }, convey.ShouldPanic)
			convey.So(func() { Stop(true) }, convey.ShouldPanic)
		})

		convey.Convey("normal", func() {
			var t1Run, t2Run int64
			// register before start
			Register("t1", TaskDefinition{Handler: testCountHandler(&t1Run)})
			// start
			StartWithOptions(testDB("TestMainProcess"), "tasks")
			// register after start
			Register("t2", TaskDefinition{Handler: testCountHandler(&t2Run)})

			err1 := Run(context.TODO(), "t1", nil)
			err2 := Run(context.TODO(), "t2", nil)
			convey.So(err1, convey.ShouldBeNil)
			convey.So(err2, convey.ShouldBeNil)

			err := Transaction(func(tx *gorm.DB) error {
				if err := RunWithTx(tx, context.TODO(), "t1", nil); err != nil {
					return err
				}
				if err := RunWithTx(tx, context.TODO(), "t2", nil); err != nil {
					return err
				}
				return nil
			})
			convey.So(err, convey.ShouldBeNil)

			Stop(true)
			convey.So(t1Run, convey.ShouldEqual, 2)
			convey.So(t2Run, convey.ShouldEqual, 2)
		})
	})
}

func TestDefaultManager(t *testing.T) {
	convey.Convey("TestDefaultManager", t, func() {
		m := DefaultManager()
		convey.So(m, convey.ShouldNotBeNil)
	})
}
