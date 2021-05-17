# GTA - Go Task Async

A lightweight and reliable asynchronous task and transaction message library for Golang.&nbsp;&nbsp;[🇺🇸English | [🇨🇳中文](README_zh-CN.md)]

[![Go Report Card](https://goreportcard.com/badge/github.com/ycydsxy/gta)](https://goreportcard.com/report/github.com/ycydsxy/gta)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/ycydsxy/gta/Go?logo=github)](https://github.com/ycydsxy/gta/actions/workflows/go.yml)
![Travis (.com)](https://img.shields.io/travis/com/ycydsxy/gta?label=test&logo=travis)
[![Coverage](https://img.shields.io/codecov/c/github/ycydsxy/gta?logo=codecov)](https://codecov.io/gh/ycydsxy/gta)
[![GitHub issues](https://img.shields.io/github/issues/ycydsxy/gta)](https://github.com/ycydsxy/gta/issues)
[![Release](https://img.shields.io/github/v/release/ycydsxy/gta.svg)](https://github.com/ycydsxy/gta/releases)
[![GitHub license](https://img.shields.io/github/license/ycydsxy/gta?color=blue)](https://github.com/ycydsxy/gta/blob/main/LICENSE)

## Overview
GTA (Go Task Async) is a lightweight and reliable asynchronous task and transaction message library for by golang. The framework has the following characteristics：
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
# Configuration

## Global optional configuration

When calling `StartWithOptions` or `NewTaskManager`, one or more optional configurations can be specified according to the incoming order. If the configuration name is `XXX`, the configuration can be specified with `WithXXX`. For example, the following code can specify that the pool size is 10 and the dry run flag is true:

```golang
gta.StartWithOptions(db, "tasks", gta.WithPoolSize(10), gta.WithDryRun(true))
```

All optional configurations and default values are as follows:

|name |type| default value | meaning |
| ------------------- | ----------------------------------------- | -------------------- | ------------------------------------------------------------ |
|Context | context. Context | context.Background() | root context, used for the framework itself|
|LoggerFactory | func(ctx context.Context) Logger | defaultLoggerFactory | log factory method for log printing|
|StorageTimeout | time.Duration | 1 week | determines how long a completed task will be cleaned up|
|InitializedTimeout | time.Duration | 5 minutes | determines how long an initialized task will be considered abnormal|
|RunningTimeout | time.Duration | 30 minutes | determines how long an ongoing task will be considered abnormal|
|WaitTimeout | time.Duration | waiting all the time | determines the longest execution time of the `Stop` function when a task is running |
|ScanInterval | time.Duration | 5 seconds | determines the speed of scanning initialized task under normal circumstances|
|InstantScanInvertal | time. Duration | 100 ms | determines the scan speed when there are unprocessed initialized tasks|
|CtxMarshaler | CtxMarshaler | defaultCtxMarshaler | determines how context is serialized|
|CheckCallback | func(logger Logger, abnormalTasks []Task) | defaultcheckcallback | determines how to handle the detected abnormal task|
|DryRun | bool | false | dry run flag is used to test and determines whether to run without relying on the database|
|PoolSize | int | math.MaxInt32 | determines how many goroutines can be used to run tasks|
## Single task definition
When calling `Register` for task registration, you need to pass in the corresponding task definition, as follows:

|name|type|default value|meaning|
| -------------------- | ------------------------------------------------------ | ---------------- | ------------------------------------------------------------ |
|Handler | func(ctx context.Context, arg interface{}) (err error) | no | required, task handler|
|ArgType | reflect.Type | nil | determines the actual type of arg in the task processing function. If it is empty, the type of arg is `map[string]interface{}` |
|CtxMarshaler | CtxMarshaler | global CtxMarshaler | determines how to serialize the context.context of a task|
|RetryTimes | int | 0 | the maximum number of retries when a task fails. Tasks exceeding this value will be marked as failed|
|RetryInterval | func(times int) time.Duration | 1 second | the interval between two retries of task execution error|
|CleanSucceeded | bool | false |whether to clear the task record immediately after the success. If so, the task record will be cleared immediately after succeeded|
|InitTimeoutSensitive | bool | false | determines whether the task is sensitive to `InitializedTimeout`. If so, it cannot be scanned and scheduled after `InitializedTimeout`|
# Frequently asked questions

## What is an abnormal task? How to detect abnormal tasks?

Abnormal tasks include tasks that have not been scheduled for a long time, tasks that have timed out, or tasks that have been aborted due to non graceful shutdown

Abnormal tasks will be detected by the built-in tasks executed regularly, and the configured `CheckCallback` will be called. By default, the number of abnormal tasks and the corresponding ID will be printed through the log

## Will the pool block when it is full?

In the current design, the steps of submitting tasks are not blocked. When the pool is full, the submitted tasks will be transferred to other instances for execution, and the scanning mechanism will be suspended; When all the instance pools are full, the tasks will be overstocked in the database

## Is there a delay in scheduling asynchronous tasks?

If it is based on the Commit Hook mechanism, there is almost no delay, such as calling `Run` in the case of sufficient co pool or calling `RunWithTx` in the built-in `Transaction`.

If the pool is full, or the `RunWithTx` is invoked in a non built `Transaction`, then the asynchronous task is scheduled based on the scan mechanism. At this time, the schedule is delayed. The delay time is the time required for all instances to compete and schedule the task, and is related to the scan interval, the  pool idle time and the backlog of tasks.

## The task consumption ability of scanning mechanism?

The maximum consumption capacity is `N*1/InstantScanInterval` tasks per second, where n is the number of instances, instantscaninterval is the fast scan interval, and the consumption capacity of a single instance is set to 10/s by default. The scheduling ability of scanning mechanism is limited, and it is an auxiliary scheduling method. Reducing `InstantScanInterval` can improve the consumption ability, but it will also increase the database pressure. Therefore, under normal circumstances, we should try to use commit hook mechanism

## How to handle abnormal and failed tasks?

Under normal circumstances, the exception and failure of a task are small probability events. If the exception and failure of a task are caused by some factors (such as external resource exception, abnormal downtime, etc.), it can be rescheduled manually with corresponding APIs provided by `TaskManager`, such as `ForceRerunTasks` and `QueryUnsuccessfulTasks`

## How to test?

You can use `WithDryRun(true)` to make the framework enter dry running mode to avoid the data impact caused by reading and writing task tables of other instances. In this mode, the framework will not read and write task tables, nor record task status and other information


## License

[MIT](https://github.com/ycydsxy/gta/blob/main/LICENSE) 
