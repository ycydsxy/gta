# GTA - Go Task Async

A lightweight and reliable asynchronous task and transaction message library for Golang.

[![Go Report Card](https://goreportcard.com/badge/github.com/ycydsxy/gta)](https://goreportcard.com/report/github.com/ycydsxy/gta)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/ycydsxy/gta/Go?logo=github)](https://github.com/ycydsxy/gta/actions/workflows/go.yml)
![Travis (.com)](https://img.shields.io/travis/com/ycydsxy/gta?label=test&logo=travis)
[![Coverage](https://img.shields.io/codecov/c/github/ycydsxy/gta?logo=codecov)](https://codecov.io/gh/ycydsxy/gta)
[![GitHub issues](https://img.shields.io/github/issues/ycydsxy/gta)](https://github.com/ycydsxy/gta/issues)
[![Release](https://img.shields.io/github/v/release/ycydsxy/gta.svg)](https://github.com/ycydsxy/gta/releases)
[![GitHub license](https://img.shields.io/github/license/ycydsxy/gta)](https://github.com/ycydsxy/gta/blob/main/LICENSE)

## Overview
GTA (go task async) is a lightweight and reliable asynchronous task and transaction message library for by golang. The framework has the following characteristics：
- High reliability: ensure the scheduling and execution of asynchronous tasks At Least Once, and the status of all submitted tasks can be traced back
- Flexible configuration: it provides a number of simple and easy-to-use optional configuration items, which can better fit the needs of different situations
- Allow to submit multiple tasks: allow to submit multiple tasks in the same transaction (it is not guaranteed that the tasks will be executed in the order of submission)
- Allow to submit nested tasks: allow to submit new asynchronous tasks among submitted tasks (ensure that tasks are executed in the order of submission)
- Multiple scheduling methods: one is low latency scheduling similar to 'Commit Hook' mechanism and the other is preemptive scheduling based on scan mechanism. The former gives priority to the current instance, while the latter's scheduling right depends on the result of multi instance competition
- Built in tasks: provide multiple built-in tasks running on this framework for abnormal task monitoring, historical task cleaning, etc
- Graceful stop: provide graceful stop mechanism, try not to let the running task be stopped violently when the instance exits
- Pooling: the bottom layer uses the goroutine pool to run asynchronous tasks, and the size of the coroutine pool can be configured
- Lightweight: external dependence has and only has [GORM](https://github.com/go-gorm/gorm) and relational database

Users can submit, schedule, execute and monitor asynchronous tasks through this framework. It relies on relational database to ensure the reliability and traceability of asynchronous tasks. It can be used in various situations that need to ensure the successful execution of tasks (try our best to ensure the success, unless the task itself or external resources are abnormal).

In addition, the framework allows asynchronous tasks to be submitted in a transaction to ensure the strong correlation between the task and the transaction. That is, if the transaction fails to roll back, the asynchronous task will not be executed. If the transaction is successfully submitted, the asynchronous task will be executed. Therefore, it is also an implementation of transaction message.

## Install
```powershell
go get -u github.com/ycydsxy/gta
```
## Getting Started
```golang
package main

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ycydsxy/gta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// database and task table(please refer to model.sql for table schema) should be prepared first
	// here is for test only, don't use in production
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic(err)
	}
	if err = db.AutoMigrate(&gta.Task{}); err != nil {
		panic(err)
	}

	// start gta
	gta.StartWithOptions(db, "tasks")
	defer gta.Stop(true)

	// register a certain async task
	gta.Register("foo_task", gta.TaskDefinition{
		Handler: func(ctx context.Context, arg interface{}) (err error) {
			time.Sleep(time.Second)
			logrus.Warn("task done")
			return nil
		},
	})

	// run simple async task
	if err := gta.Run(context.TODO(), "foo_task", nil); err != nil {
		logrus.Errorf("error in async task, err: %v", err)
	}

	// run async task in transaction
	if err := gta.Transaction(func(tx *gorm.DB) error {
		if err := gta.RunWithTx(tx, context.TODO(), "foo_task", nil); err != nil {
			return err
		}
		return nil
	}); err != nil {
		logrus.Errorf("error in transaction with async task, err: %v", err)
	}
}
```

## License
[MIT](https://github.com/ycydsxy/gta/blob/main/LICENSE) 
