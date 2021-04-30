package gta_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	. "gta"
)

const (
	testNormalTask TaskKey = "test_normal_task"
	testPanicTask  TaskKey = "test_panic_task"

	testCtxUserIDKey       = "test_userID"
	testCtxTenantIDKey     = "test_tenantID"
	testCtxRequestIDKey    = "test_requestID"
	testCtxTaskContextCKey = "test_task_c"
)

func init() {
	Register(testNormalTask, TaskDefinition{
		Handler: func(ctx context.Context, arg interface{}) (err error) {
			return normalTestTaskFunc(ctx, arg.(*testTaskArg))
		},
		ArgType:        reflect.TypeOf(&testTaskArg{}),
		CtxMarshaler:   taskCtxMarshaler{},
		RetryTimes:     1,
		CleanSucceeded: true,
	})

	Register(testPanicTask, TaskDefinition{
		Handler: func(ctx context.Context, arg interface{}) (err error) {
			panic(arg)
		},
	})
}

func TestRegister(t *testing.T) {
	convey.Convey("TestRegister", t, func() {
		convey.So(func() { Register("", TaskDefinition{}) }, convey.ShouldPanic)
	})
}

func TestMainProcess(t *testing.T) {
	db, err := gorm.Open(mysql.Open("root@(127.0.0.1:3306)/test_db?charset=utf8&parseTime=True&loc=UTC"))
	if err != nil {
		panic(err)
	}

	// normal init
	StartWithOptions(db.Debug(), "async_task_test", WithConfig(TaskConfig{
		LoggerFactory:      loggerFactory,
		Context:            rootContext(),
		CtxMarshaler:       testCtxMarshaler{},
		ScanInterval:       time.Second,
		InitializedTimeout: time.Second * 5,
		RunningTimeout:     time.Second * 8,
		StorageTimeout:     time.Second * 30,
		PoolSize:           5,
	}))

	// biz process
	convey.Convey("TestMainProcess", t, func() {
		convey.Convey("test error", func() {
			convey.Convey("argument mismatch", func() {
				taskCtx := mockTaskContext("", "", "", "")
				var mismatchArgument = "mismatch"
				err := Run(taskCtx, testNormalTask, &mismatchArgument)
				convey.So(err, convey.ShouldNotBeNil)
			})

			convey.Convey("handler panic", func() {
				taskCtx := mockTaskContext("", "", "", "")
				err := Run(taskCtx, testPanicTask, testTaskArg{A: 10000, B: 10000})
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("test normal", func() {
			convey.Convey("no transaction funcs", func() {
				for i := 5; i < 10; i++ {
					taskCtx := mockTaskContext(fmt.Sprintf("req_1000000%d", i), strconv.Itoa(i+10000),
						strconv.Itoa(i-10000), strconv.Itoa(2*i))
					if err := Run(taskCtx, testNormalTask, &testTaskArg{A: i, B: 2 * i}); err != nil {
						convey.So(err, convey.ShouldBeNil)
					}
				}
			})

			convey.Convey("transaction funcs", func() {
				fa := func(tx *gorm.DB) error {
					fmt.Println("fa function succeeded")
					return nil
				}

				fb := func(tx *gorm.DB) error {
					fmt.Println("fb function failed")
					return errors.New("fb failed")
				}

				// no task
				err := Transaction(func(tx *gorm.DB) error {
					return fa(tx)
				})
				convey.So(err, convey.ShouldBeNil)

				err = Transaction(func(tx *gorm.DB) error {
					return errors.New("transaction error")
				})
				convey.So(err, convey.ShouldNotBeNil)

				// single no-builtin transaction, tf succeeded, hander successded
				taskCtx := mockTaskContext("req_1000000fa_t", "fa_t", "fa_u", "fa_c")
				err = db.Transaction(func(tx *gorm.DB) error {
					if err := fa(tx); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 0, B: 0}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)

				// single transaction funcs(tf), tf succeeded, hander successded
				taskCtx = mockTaskContext("req_1000000fa_t", "fa_t", "fa_u", "fa_c")
				err = Transaction(func(tx *gorm.DB) error {
					if err := fa(tx); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 0, B: 0}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)

				// multiple tf, tf succeeded, hander failed
				taskCtx = mockTaskContext("req_1000000faa_t", "faa_t", "faa_u", "faa_c")
				err = Transaction(func(tx *gorm.DB) error {
					if err := fa(tx); err != nil {
						return err
					}
					if err := fa(tx); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 1, B: 2}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)

				// signle tf, tf failed
				taskCtx = mockTaskContext("req_1000000fb_t", "fb_t", "fb_u", "fb_c")
				err = Transaction(func(tx *gorm.DB) error {
					if err := fb(tx); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 2, B: 4}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldNotBeNil)

				// multiple tf, tf failed
				taskCtx = mockTaskContext("req_1000000fab_t", "fab_t", "fab_u", "fab_c")
				err = Transaction(func(tx *gorm.DB) error {
					if err := fa(tx); err != nil {
						return err
					}
					if err := fb(tx); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 3, B: 6}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldNotBeNil)
			})

			convey.Convey("wait funcs", func() {
				taskCtx := mockTaskContext("req_1000000wait1", "wait1", "wait1", "wait1")
				if err := Run(taskCtx, testNormalTask, &testTaskArg{A: 20, B: 42}); err != nil {
					convey.So(err, convey.ShouldBeNil)
				}

				taskCtx = mockTaskContext("req_1000000wait2", "wait2", "wait2", "wait2")
				if err := Run(taskCtx, testNormalTask, &testTaskArg{A: 20, B: 60}); err != nil {
					convey.So(err, convey.ShouldBeNil)
				}
			})
		})
	})

	time.Sleep(time.Second * 40)

	go func() {
		convey.Convey("TestProcessAfterClean", t, func() {
			time.Sleep(time.Second)
			convey.Convey("normal", func() {
				taskCtx := mockTaskContext("ctx_done1", "ctx_done1", "ctx_done1", "ctx_done1")
				err := Run(taskCtx, testNormalTask, &testTaskArg{A: 2, B: 4})
				convey.So(err, convey.ShouldBeNil)
			})
			convey.Convey("normal transaction", func() {
				taskCtx := mockTaskContext("ctx_done2", "ctx_done2", "ctx_done2", "ctx_done2")
				err := Transaction(func(tx *gorm.DB) error {
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 2, B: 4}); err != nil {
						return err
					}
					if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 3, B: 6}); err != nil {
						return err
					}
					return nil
				})
				convey.So(err, convey.ShouldBeNil)
			})
		})
	}()

	Stop(false)

	convey.Convey("TestDevops", t, func() {
		convey.Convey("TestQueryUnsuccessfulTasks", func() {
			res, err := QueryUnsuccessfulTasks()
			convey.So(err, convey.ShouldBeNil)
			convey.So(res, convey.ShouldNotBeEmpty)
		})

		convey.Convey("TestForceRerunTask", func() {
			convey.Convey("normal", func() {
				err := ForceRerunTask(10003, TaskStatusFailed)
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("error", func() {
				err := ForceRerunTask(0, TaskStatusFailed)
				convey.So(err, convey.ShouldNotBeNil)
			})
		})
	})
}

func loggerFactory(ctx context.Context) Logger {
	fields := logrus.Fields{
		"psm": "test_psm",
	}
	if userID := ctx.Value(testCtxUserIDKey); userID != nil {
		fields[testCtxUserIDKey] = userID
	}
	if tenantID := ctx.Value(testCtxTenantIDKey); tenantID != nil {
		fields[testCtxTenantIDKey] = tenantID
	}
	if requestID := ctx.Value(testCtxRequestIDKey); requestID != nil {
		fields[testCtxRequestIDKey] = requestID
	}
	return logrus.WithFields(fields)
}

func mockTaskContext(requestID string, tenantID string, userID string, c string) context.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request, _ = http.NewRequest("POST", "/", nil)
	ctx.Set(testCtxRequestIDKey, requestID)
	ctx.Set(testCtxTenantIDKey, tenantID)
	ctx.Set(testCtxUserIDKey, userID)
	ctx.Set(testCtxTaskContextCKey, c)
	return ctx
}

func normalTestTaskFunc(ctx context.Context, req *testTaskArg) error {
	logger := loggerFactory(ctx)
	logger.Infof("[normalTestTaskFunc] inside handler, start, req[%+v], ctx_param[%v]", req,
		ctx.Value(testCtxTaskContextCKey))

	// do business processes
	time.Sleep(time.Second * time.Duration(req.B))
	if req.A%5 == 1 { // mock error
		return errors.New("test error")
	}

	logger.Infof("[normalTestTaskFunc] inside handler, end, req[%+v], ctx_param[%v]", req,
		ctx.Value(testCtxTaskContextCKey))
	return nil
}

type testTaskArg struct {
	A int
	B int
}

func rootContext() context.Context {
	ctx := context.Background()
	return context.WithValue(ctx, testCtxRequestIDKey, "root_request_id")
}

type testCtxMarshaler struct{}

func (t testCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	requestID := ctx.Value(testCtxRequestIDKey)
	if requestID != nil {
		requestID = "null_requestID"
	}
	return []byte(requestID.(string)), nil
}

func (t testCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	requestID := string(bytes)
	return context.WithValue(context.Background(), testCtxRequestIDKey, requestID), nil
}

type taskCtxMarshaler struct {
	ReqestID string `json:"reqest_id"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	C        string `json:"c"`
}

func (t taskCtxMarshaler) MarshalCtx(ctx context.Context) ([]byte, error) {
	c := taskCtxMarshaler{}
	if requestID := ctx.Value(testCtxRequestIDKey); requestID != nil {
		c.ReqestID = requestID.(string)
	}
	if cv := ctx.Value(testCtxTaskContextCKey); cv != nil {
		c.C = cv.(string)
	}
	if tenantID, err := (tenantIDValuer{}).TenantID(ctx); err != nil {
		return nil, err
	} else {
		c.TenantID = tenantID
	}
	if userID, err := (userIDValuer{}).UserID(ctx); err != nil {
		return nil, err
	} else {
		c.UserID = userID
	}
	return json.Marshal(c)
}

func (t taskCtxMarshaler) UnmarshalCtx(bytes []byte) (context.Context, error) {
	var c taskCtxMarshaler
	if err := json.Unmarshal(bytes, &c); err != nil {
		return nil, err
	}
	return mockTaskContext(c.ReqestID, c.TenantID, c.UserID, c.C), nil
}

type tenantIDValuer struct{}

func (t tenantIDValuer) TenantID(ctx context.Context) (string, error) {
	tenantID := ctx.Value(testCtxTenantIDKey)
	if tenantID == nil {
		tenantID = "null"
	}
	return tenantID.(string), nil
}

type userIDValuer struct{}

func (t userIDValuer) UserID(ctx context.Context) (string, error) {
	userID := ctx.Value(testCtxUserIDKey)
	if userID == nil {
		userID = "null"
	}
	return userID.(string), nil
}

//
// func TestDryRun(t *testing.T) {
// 	tc.DryRun = true
// 	convey.Convey("TestDryRun", t, func() {
// 		convey.Convey("no transaction funcs", func() {
// 			convey.Convey("normal", func() {
// 				i := 0
// 				taskCtx := mockTaskContext(fmt.Sprintf("req_1000000%d", i), strconv.Itoa(i+10000),
// 					strconv.Itoa(i-10000), strconv.Itoa(2*i))
// 				if err := Run(taskCtx, testNormalTask, &testTaskArg{A: i, B: 2 * i}); err != nil {
// 					convey.So(err, convey.ShouldNotBeNil)
// 				}
// 			})
//
// 			convey.Convey("error", func() {
// 				i := 1
// 				taskCtx := mockTaskContext(fmt.Sprintf("req_1000000%d", i), strconv.Itoa(i+10000),
// 					strconv.Itoa(i-10000), strconv.Itoa(2*i))
// 				if err := Run(taskCtx, testNormalTask, &testTaskArg{A: i, B: 2 * i}); err != nil {
// 					convey.So(err, convey.ShouldNotBeNil)
// 				}
// 			})
// 		})
//
// 		convey.Convey("transaction funcs", func() {
// 			fa := func(tx *gorm.DB) error {
// 				fmt.Println("fa function succeeded")
// 				return nil
// 			}
//
// 			fb := func(tx *gorm.DB) error {
// 				fmt.Println("fb function failed")
// 				return errors.New("fb failed")
// 			}
//
// 			// no task
// 			err := Transaction(func(tx *gorm.DB) error {
// 				return fa(tx)
// 			})
// 			convey.So(err, convey.ShouldBeNil)
//
// 			err = Transaction(func(tx *gorm.DB) error {
// 				return errors.New("transaction error")
// 			})
// 			convey.So(err, convey.ShouldNotBeNil)
//
// 			// single transaction funcs(tf), tf succeeded, hander successded
// 			taskCtx := mockTaskContext("req_1000000fa_t", "fa_t", "fa_u", "fa_c")
// 			err = Transaction(func(tx *gorm.DB) error {
// 				if err := fa(tx); err != nil {
// 					return err
// 				}
// 				if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 0, B: 0}); err != nil {
// 					return err
// 				}
// 				return nil
// 			})
// 			convey.So(err, convey.ShouldBeNil)
//
// 			// multiple tf, tf succeeded, hander failed
// 			taskCtx = mockTaskContext("req_1000000faa_t", "faa_t", "faa_u", "faa_c")
// 			err = Transaction(func(tx *gorm.DB) error {
// 				if err := fa(tx); err != nil {
// 					return err
// 				}
// 				if err := fa(tx); err != nil {
// 					return err
// 				}
// 				if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 1, B: 2}); err != nil {
// 					return err
// 				}
// 				return nil
// 			})
// 			convey.So(err, convey.ShouldBeNil)
//
// 			// signle tf, tf failed
// 			taskCtx = mockTaskContext("req_1000000fb_t", "fb_t", "fb_u", "fb_c")
// 			err = Transaction(func(tx *gorm.DB) error {
// 				if err := fb(tx); err != nil {
// 					return err
// 				}
// 				if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 2, B: 4}); err != nil {
// 					return err
// 				}
// 				return nil
// 			})
// 			convey.So(err, convey.ShouldNotBeNil)
//
// 			// multiple tf, tf failed
// 			taskCtx = mockTaskContext("req_1000000fab_t", "fab_t", "fab_u", "fab_c")
// 			err = Transaction(func(tx *gorm.DB) error {
// 				if err := fa(tx); err != nil {
// 					return err
// 				}
// 				if err := fb(tx); err != nil {
// 					return err
// 				}
// 				if err := RunWithTx(tx, taskCtx, testNormalTask, &testTaskArg{A: 3, B: 6}); err != nil {
// 					return err
// 				}
// 				return nil
// 			})
// 			convey.So(err, convey.ShouldNotBeNil)
// 		})
//
// 	})
// }
