package gta

import (
	"context"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestTaskDefinition_init(t *testing.T) {
	convey.Convey("TestTaskDefinition_init", t, func() {
		convey.Convey("normal", func() {
			taskDef := &TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }}
			err := taskDef.init("key")
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("nil handler", func() {
			taskDef := &TaskDefinition{}
			err := taskDef.init("key")
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("built in task but has no taskID", func() {
			taskDef := &TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }, builtin: true}
			err := taskDef.init("key")
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("built in task but has no loop interval", func() {
			taskDef := &TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }, builtin: true, taskID: 1}
			err := taskDef.init("key")
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("built in task but has no argument", func() {
			taskDef := &TaskDefinition{Handler: func(ctx context.Context, arg interface{}) (err error) { return nil }, builtin: true, taskID: 1, loopInterval: 1}
			err := taskDef.init("key")
			convey.So(err, convey.ShouldNotBeNil)
		})

	})
}

func TestTaskDefinition_ctxMarshaler(t *testing.T) {
	convey.Convey("TestTaskDefinition_ctxMarshaler", t, func() {
		convey.Convey("empty ctxMarshal in taskDef", func() {
			taskDef := &TaskDefinition{}
			cm := taskDef.ctxMarshaler(&defaultCtxMarshaler{})
			convey.So(cm, convey.ShouldNotBeNil)
		})

		convey.Convey("specify ctxMarshal in taskDef", func() {
			taskDef := &TaskDefinition{CtxMarshaler: &defaultCtxMarshaler{}}
			cm := taskDef.ctxMarshaler(nil)
			convey.So(cm, convey.ShouldNotBeNil)
		})
	})
}

func TestTaskDefinition_retryInterval(t *testing.T) {
	convey.Convey("TestTaskDefinition_retryInterval", t, func() {
		convey.Convey("empty retryInterval in taskDef", func() {
			taskDef := &TaskDefinition{}
			r := taskDef.retryInterval(1)
			convey.So(r, convey.ShouldEqual, time.Second)
		})

		convey.Convey("specify retryInterval in taskDef", func() {
			taskDef := &TaskDefinition{RetryInterval: func(times int) time.Duration { return time.Hour }}
			r := taskDef.retryInterval(1)
			convey.So(r, convey.ShouldEqual, time.Hour)
		})
	})
}
