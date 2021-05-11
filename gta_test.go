package gta

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
