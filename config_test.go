package gta

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func Test_Config_init(t *testing.T) {
	convey.Convey("Test_Config_init", t, func() {
		convey.Convey("normal process", func() {
			tc := Config{DB: defaultDB, TableName: defaultTableName}
			convey.So(tc.init(), convey.ShouldBeNil)
		})

		convey.Convey("empty table name", func() {
			tc := Config{}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("empty db factory", func() {
			tc := Config{TableName: defaultTableName}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid running timeout", func() {
			tc := Config{DB: defaultDB, TableName: defaultTableName, RunningTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid initialized timeout", func() {
			tc := Config{DB: defaultDB, TableName: defaultTableName, InitializedTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid scan interval", func() {
			tc := Config{DB: defaultDB, TableName: defaultTableName, ScanInterval: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

	})
}
