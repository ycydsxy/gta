package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func Test_panicHandler(t *testing.T) {
	convey.Convey("Test_panicHandler", t, func() {
		convey.So(func() { defer panicHandler(); panic("test") }, convey.ShouldNotPanic)
	})
}

func Test_randomInterval(t *testing.T) {
	convey.Convey("Test_randomInterval", t, func() {
		interval := randomInterval(time.Second)
		convey.So(interval, convey.ShouldBeGreaterThan, time.Second)
		convey.So(interval, convey.ShouldBeLessThan, time.Duration(float64(time.Second)*(1+randomIntervalFactor)))
	})
}
