package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
)

func Test_Config_init(t *testing.T) {
	defaultDB := &gorm.DB{}
	defaultTable := "tasks"

	convey.Convey("Test_Config_init", t, func() {
		convey.Convey("normal process", func() {
			tc := TaskConfig{DB: defaultDB, Table: defaultTable}
			convey.So(tc.init(), convey.ShouldBeNil)
		})

		convey.Convey("empty db factory", func() {
			tc := TaskConfig{}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("empty table name", func() {
			tc := TaskConfig{DB: defaultDB}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid running timeout", func() {
			tc := TaskConfig{DB: defaultDB, Table: defaultTable, RunningTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid initialized timeout", func() {
			tc := TaskConfig{DB: defaultDB, Table: defaultTable, InitializedTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid scan interval", func() {
			tc := TaskConfig{DB: defaultDB, Table: defaultTable, ScanInterval: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid instant scan interval", func() {
			tc := TaskConfig{DB: defaultDB, Table: defaultTable, InstantScanInvertal: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

	})
}
