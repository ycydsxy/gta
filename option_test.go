package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func Test_newOptions(t *testing.T) {
	defaultDB := &gorm.DB{}
	defaultTable := "tasks"

	convey.Convey("Test_newOptions", t, func() {
		convey.Convey("normal process", func() {
			_, err := newOptions(defaultDB, defaultTable)
			convey.So(err, convey.ShouldBeNil)
		})

		convey.Convey("nil db", func() {
			_, err := newOptions(defaultDB, defaultTable, withDB(nil))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("empty table name", func() {
			_, err := newOptions(defaultDB, defaultTable, withTable(""))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("nil logger factory", func() {
			_, err := newOptions(defaultDB, defaultTable, WithLoggerFactory(nil))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("nil ctxMashaler", func() {
			_, err := newOptions(defaultDB, defaultTable, WithCtxMarshaler(nil))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid running timeout", func() {
			_, err := newOptions(defaultDB, defaultTable, WithRunningTimeout(time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)

			_, err = newOptions(defaultDB, defaultTable, WithRunningTimeout(-time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid initialized timeout", func() {
			_, err := newOptions(defaultDB, defaultTable, WithInitializedTimeout(time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid scan interval", func() {
			_, err := newOptions(defaultDB, defaultTable, WithScanInterval(time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid instant scan interval", func() {
			_, err := newOptions(defaultDB, defaultTable, WithInstantScanInterval(time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid wait timeout", func() {
			_, err := newOptions(defaultDB, defaultTable, WithWaitTimeout(-time.Hour*365))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("invalid pool size", func() {
			_, err := newOptions(defaultDB, defaultTable, WithPoolSize(-1))
			convey.So(err, convey.ShouldNotBeNil)
		})

		convey.Convey("nil task register", func() {
			_, err := newOptions(defaultDB, defaultTable, withTaskRegister(nil))
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
