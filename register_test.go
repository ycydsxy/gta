package gta

import (
	"context"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func Test_taskRegister_Register(t *testing.T) {
	convey.Convey("Test_taskRegister_Register", t, func() {
		convey.Convey("invalid length", func() {
			tr := taskRegisterImp{}
			err := tr.Register(TaskKey(strings.Repeat("1", 65)), TaskDefinition{
				Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			})
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("register twice", func() {
			tr := taskRegisterImp{}
			err := tr.Register("test", TaskDefinition{
				Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			})
			convey.So(err, convey.ShouldBeNil)
			err = tr.Register("test", TaskDefinition{
				Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			})
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("definition init error", func() {
			convey.Convey("nil handler", func() {
				tr := taskRegisterImp{}
				err := tr.Register("test", TaskDefinition{})
				convey.So(err, convey.ShouldNotBeNil)
			})

			convey.Convey("builtin tasks", func() {
				convey.Convey("zero task id", func() {
					tr := taskRegisterImp{}
					err := tr.Register("test", TaskDefinition{builtin: true})
					convey.So(err, convey.ShouldNotBeNil)
				})

				convey.Convey("nil loop interval factory", func() {
					tr := taskRegisterImp{}
					err := tr.Register("test", TaskDefinition{builtin: true, taskID: 3})
					convey.So(err, convey.ShouldNotBeNil)
				})

				convey.Convey("nil argument factory", func() {
					tr := taskRegisterImp{}
					err := tr.Register("test", TaskDefinition{builtin: true, taskID: 3, loopInterval: 1})
					convey.So(err, convey.ShouldNotBeNil)
				})
			})
		})
	})
}

func Test_taskRegister_GetDefinition(t *testing.T) {
	convey.Convey("Test_taskRegister_GetDefinition", t, func() {
		tr := taskRegisterImp{}
		_, err := tr.GetDefinition("not exist")
		convey.So(err, convey.ShouldNotBeNil)
	})
}
