package gta

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func Test_taskRegisterImp_Register(t *testing.T) {
	convey.Convey("Test_taskRegister_Register", t, func() {
		convey.Convey("normal", func() {
			tr := taskRegisterImp{}
			err := tr.Register("key", TaskDefinition{
				Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			})
			convey.So(err, convey.ShouldBeNil)
		})

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

func Test_taskRegisterImp_GetDefinition(t *testing.T) {
	convey.Convey("Test_taskRegister_GetDefinition", t, func() {
		tr := taskRegisterImp{}
		convey.Convey("normal", func() {
			_ = tr.Register("key", TaskDefinition{
				Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			})
			taskDef, err := tr.GetDefinition("key")
			convey.So(err, convey.ShouldBeNil)
			convey.So(taskDef, convey.ShouldNotBeNil)
		})
		convey.Convey("not exist key", func() {
			_, err := tr.GetDefinition("key")
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}

func Test_taskRegisterImp_GroupKeysByInitTimeoutSensitivity(t *testing.T) {
	convey.Convey("Test_taskRegisterImp_GroupKeysByInitTimeoutSensitivity", t, func() {
		tr := taskRegisterImp{}
		_ = tr.Register("key1", TaskDefinition{
			Handler:              func(ctx context.Context, arg interface{}) (err error) { return nil },
			InitTimeoutSensitive: false,
		})
		_ = tr.Register("key2", TaskDefinition{
			Handler:              func(ctx context.Context, arg interface{}) (err error) { return nil },
			InitTimeoutSensitive: true,
		})
		sensitiveKeys, insensitiveKeys := tr.GroupKeysByInitTimeoutSensitivity()
		convey.So(sensitiveKeys, convey.ShouldHaveLength, 1)
		convey.So(insensitiveKeys, convey.ShouldHaveLength, 1)
	})
}

func Test_taskRegisterImp_GetBuiltInKeys(t *testing.T) {
	convey.Convey("Test_taskRegisterImp_GroupKeysByInitTimeoutSensitivity", t, func() {
		tr := taskRegisterImp{}
		_ = tr.Register("key1", TaskDefinition{
			Handler: func(ctx context.Context, arg interface{}) (err error) { return nil },
			builtin: false,
		})
		_ = tr.Register("key2", TaskDefinition{
			Handler:      func(ctx context.Context, arg interface{}) (err error) { return nil },
			builtin:      true,
			taskID:       1,
			loopInterval: time.Second,
			argument:     0,
		})
		res := tr.GetBuiltInKeys()
		convey.So(res, convey.ShouldHaveLength, 1)
	})
}
