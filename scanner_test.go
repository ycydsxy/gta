package gta

import (
	"testing"

	"github.com/panjf2000/ants/v2"
	"github.com/smartystreets/goconvey/convey"
)

func Test_taskScannerImp_claimInitializedTask(t *testing.T) {
	convey.Convey("Test_taskScannerImp_claimInitializedTask", t, func() {
		convey.Convey("ctx cancelled", func() {
			tc, _ := newConfig(testDB("Test_taskScannerImp_claimInitializedTask"), "tasks")
			tr := &taskRegisterImp{}
			tdal := &taskDALImp{config: tc}
			tscn := &taskScannerImp{config: tc, register: tr, dal: tdal}
			_ = tr.Register("t1", TaskDefinition{Handler: testWrappedHandler()})
			_ = tdal.Create(tc.db(), &Task{TaskKey: "t1", TaskStatus: TaskStatusInitialized})
			tc.cancel()
			task, err := tscn.claimInitializedTask()
			convey.So(err, convey.ShouldBeNil)
			convey.So(task, convey.ShouldBeNil)
		})

		convey.Convey("error", func() {
			tc, _ := newConfig(testDB("Test_taskScannerImp_claimInitializedTask"), "not exist")
			tr := &taskRegisterImp{}
			tdal := &taskDALImp{config: tc}
			tscn := &taskScannerImp{config: tc, register: tr, dal: tdal}
			_, err := tscn.claimInitializedTask()
			convey.So(err, convey.ShouldNotBeNil)
		})

	})
}

func Test_taskScannerImp_scanAndSchedule(t *testing.T) {
	convey.Convey("Test_taskScannerImp_scanAndSchedule", t, func() {
		convey.Convey("error", func() {
			tc, _ := newConfig(testDB("Test_taskScannerImp_scanAndSchedule"), "not exist")
			tr := &taskRegisterImp{}
			tdal := &taskDALImp{config: tc}
			pool, _ := ants.NewPool(1)
			tsch := &taskSchedulerImp{pool: pool}
			tscn := &taskScannerImp{config: tc, register: tr, dal: tdal, scheduler: tsch}
			convey.So(func() { tscn.scanAndSchedule() }, convey.ShouldNotPanic)
		})
	})
}
