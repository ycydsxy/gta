package gta

import (
	"reflect"
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func Test_taskMonitorImp_monitorBuiltinTask(t *testing.T) {
	convey.Convey("Test_taskMonitorImp_monitorBuiltinTask", t, func() {
		convey.Convey("error", func() {
			convey.Convey("assemble task error", func() {
				tc, _ := newOptions(&gorm.DB{}, "tasks")
				mon := &taskMonitorImp{options: tc, assembler: &taskAssemblerImp{options: tc}}
				convey.So(func() { mon.monitorBuiltinTask(&TaskDefinition{ArgType: reflect.TypeOf(""), argument: 0}) }, convey.ShouldNotPanic)
			})
			convey.Convey("dal error", func() {
				tc, _ := newOptions(testDB("Test_taskMonitorImp_monitorBuiltinTask"), "not exist")
				mon := &taskMonitorImp{options: tc, assembler: &taskAssemblerImp{options: tc}, dal: &taskDALImp{options: tc}}
				convey.So(func() { mon.monitorBuiltinTask(&TaskDefinition{ArgType: reflect.TypeOf(""), argument: ""}) }, convey.ShouldNotPanic)
			})
		})
	})
}
