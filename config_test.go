package gta

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/smartystreets/goconvey/convey"
)

func Test_Config_init(t *testing.T) {
	convey.Convey("Test_Config_init", t, func() {
		convey.Convey("empty table name", func() {
			tc := Config{}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("empty db factory", func() {
			tc := Config{TableName: "test table"}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid running timeout", func() {
			tc := Config{TableName: "test table", DBFactory: func() *gorm.DB {
				return nil
			}, RunningTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid initialized timeout", func() {
			tc := Config{TableName: "test table", DBFactory: func() *gorm.DB {
				return nil
			}, InitializedTimeout: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("invalid scan interval", func() {
			tc := Config{TableName: "test table", DBFactory: func() *gorm.DB {
				return nil
			}, ScanInterval: time.Hour * 365}
			convey.So(tc.init(), convey.ShouldNotBeNil)
		})

		convey.Convey("normal process", func() {
			tc := Config{TableName: "test table", DBFactory: func() *gorm.DB {
				return nil
			}}
			convey.So(tc.init(), convey.ShouldBeNil)
		})
	})
}
