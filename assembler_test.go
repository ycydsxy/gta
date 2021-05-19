package gta

import (
	"context"
	"reflect"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

type testErrCtxMarshaler struct{}

func (e *testErrCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	return nil, ErrUnexpected
}

func (e *testErrCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	return nil, ErrUnexpected
}

func Test_taskAssemblerImp_AssembleTask(t *testing.T) {
	convey.Convey("Test_taskAssemblerImp_AssembleTask", t, func() {
		tass := taskAssemblerImp{options: &options{ctxMarshaler: &defaultCtxMarshaler{}}}
		convey.Convey("normal", func() {
			convey.Convey("nil arg type", func() {
				task, err := tass.AssembleTask(context.TODO(), &TaskDefinition{}, nil)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
				convey.So(task.ID, convey.ShouldBeZeroValue)
			})

			convey.Convey("normal arg type", func() {
				task, err := tass.AssembleTask(context.TODO(), &TaskDefinition{ArgType: reflect.TypeOf(0)}, 5)
				convey.So(err, convey.ShouldBeNil)
				convey.So(task, convey.ShouldNotBeNil)
				convey.So(task.ID, convey.ShouldBeZeroValue)
			})
		})

		convey.Convey("arg mismatch", func() {
			task, err := tass.AssembleTask(context.TODO(), &TaskDefinition{ArgType: reflect.TypeOf(0)}, "0")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(task, convey.ShouldBeNil)
		})

		convey.Convey("arg cannot marshal", func() {
			task, err := tass.AssembleTask(context.TODO(), &TaskDefinition{}, func() {})
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(task, convey.ShouldBeNil)
		})

		convey.Convey("ctx cannot marshal", func() {
			task, err := tass.AssembleTask(context.TODO(), &TaskDefinition{CtxMarshaler: &testErrCtxMarshaler{}}, "0")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(task, convey.ShouldBeNil)
		})
	})
}

func Test_taskAssemblerImp_DisassembleTask(t *testing.T) {
	convey.Convey("Test_taskAssemblerImp_DisassembleTask", t, func() {
		tass := taskAssemblerImp{options: &options{ctxMarshaler: &defaultCtxMarshaler{}}}
		convey.Convey("normal", func() {
			convey.Convey("nil arg type", func() {
				taskDef := &TaskDefinition{}
				task, _ := tass.AssembleTask(context.TODO(), taskDef, 5)
				ctx, arg, err := tass.DisassembleTask(taskDef, task)
				convey.So(err, convey.ShouldBeNil)
				convey.So(ctx, convey.ShouldNotBeNil)
				convey.So(arg, convey.ShouldNotBeNil)
				convey.So(arg.(float64), convey.ShouldEqual, float64(5))
			})

			convey.Convey("normal arg type", func() {
				taskDef := &TaskDefinition{ArgType: reflect.TypeOf(0)}
				task, _ := tass.AssembleTask(context.TODO(), taskDef, 5)
				ctx, arg, err := tass.DisassembleTask(taskDef, task)
				convey.So(err, convey.ShouldBeNil)
				convey.So(ctx, convey.ShouldNotBeNil)
				convey.So(arg, convey.ShouldNotBeNil)
				convey.So(arg.(int), convey.ShouldEqual, 5)
			})
		})

		convey.Convey("unmarshal ctx error", func() {
			taskDef := &TaskDefinition{ArgType: reflect.TypeOf(0)}
			task, _ := tass.AssembleTask(context.TODO(), taskDef, 5)
			taskDef.CtxMarshaler = &testErrCtxMarshaler{}
			ctx, arg, err := tass.DisassembleTask(taskDef, task)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(ctx, convey.ShouldBeNil)
			convey.So(arg, convey.ShouldBeNil)
		})

		convey.Convey("unmarshal arg error", func() {
			taskDef := &TaskDefinition{ArgType: reflect.TypeOf(0)}
			task, _ := tass.AssembleTask(context.TODO(), taskDef, 5)
			task.Argument = []byte{0, 2, 1}
			ctx, arg, err := tass.DisassembleTask(taskDef, task)
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(ctx, convey.ShouldBeNil)
			convey.So(arg, convey.ShouldBeNil)
		})
	})
}
