# GTA - Go Task Async

一个轻量的可靠异步任务和事务消息框架&nbsp;&nbsp;[[🇺🇸English](https://github.com/ycydsxy/gta#readme) | 🇨🇳中文]

[![Go Report Card](https://goreportcard.com/badge/github.com/ycydsxy/gta)](https://goreportcard.com/report/github.com/ycydsxy/gta)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/ycydsxy/gta/Go?logo=github)](https://github.com/ycydsxy/gta/actions/workflows/go.yml)
![Travis (.com)](https://img.shields.io/travis/com/ycydsxy/gta?label=test&logo=travis)
[![Coverage](https://img.shields.io/codecov/c/github/ycydsxy/gta?logo=codecov)](https://codecov.io/gh/ycydsxy/gta)
[![GitHub issues](https://img.shields.io/github/issues/ycydsxy/gta)](https://github.com/ycydsxy/gta/issues)
[![Release](https://img.shields.io/github/v/release/ycydsxy/gta.svg)](https://github.com/ycydsxy/gta/releases)
[![GitHub license](https://img.shields.io/github/license/ycydsxy/gta?color=blue)](https://github.com/ycydsxy/gta/blob/main/LICENSE)

## 简介
GTA(Go Task Async) 是一个 Golang 实现的轻量可靠异步任务和事务消息框架，该框架有如下一些特性：
- 高可靠性：保证异步任务 At Least Once 级别的调度和执行，所有提交的任务状态可追溯
- 灵活的配置：提供了多个简单易用的可选配置项，能够较好地贴合不同场景的需求
- 允许提交多个任务：允许在同一个事务中提交多个任务（不保证任务按提交的顺序执行）
- 允许提交嵌套任务：允许在提交的任务中提交新的异步任务（保证任务按提交的顺序执行）
- 多种调度方式：提供类似 Commit Hook 机制的低延时调度和基于扫描机制的抢占式调度两种调度方式，前者优先在当前实例上进行调度，后者调度权取决于多实例竞争的结果
- 内置任务：提供多个运行在本框架上的内置任务，用来进行异常任务监控、历史任务清理等工作
- 优雅停止：提供优雅停止机制，在实例退出时尽量不让运行中的任务被暴力中止
- 池化：底层使用协程池运行异步任务，协程池大小可配置
- 轻量：外部依赖有且仅有 [GORM](https://github.com/go-gorm/gorm) 和关系型数据库

用户能够通过它进行异步任务的提交、调度、执行和监控，其依赖关系型数据库来保证异步任务的可靠性和可追溯性，能够被运用在需要保证任务成功执行（尽最大努力保证成功，除非任务本身或外部资源异常）的各种场景

另外，该框架允许将异步任务放在某个事务中提交，以保证该任务和事务的强相关性，即如果事务失败回滚则该异步任务不执行，如果事务成功提交则该异步任务执行，故其也是事务消息的一种实现方式

## 安装
```powershell
go get -u github.com/ycydsxy/gta
```
## 使用
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
	// 首先应准备好数据库和相关表（表结构请参阅model.sql），此处代码仅用于测试，不要将其运用于生产环境中
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic(err)
	}
	if err = db.AutoMigrate(&gta.Task{}); err != nil {
		panic(err)
	}

	// 启动gta
	gta.StartWithOptions(db, "tasks")
	defer gta.Stop(true)

	// 注册一个任务
	gta.Register("foo_task", gta.TaskDefinition{
		Handler: func(ctx context.Context, arg interface{}) (err error) {
			time.Sleep(time.Second)
			logrus.Warn("task done")
			return nil
		},
	})

	// 运行任务
	if err := gta.Run(context.TODO(), "foo_task", nil); err != nil {
		logrus.Errorf("error in async task, err: %v", err)
	}

	// 在事务中运行任务
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
# 配置项
## 全局可选配置
在调用 `StartWithOptions` 或者 `NewTaskManager` 时，允许指定一个或者多个可选配置，框架将按照配置的传入顺序进行应用。若配置名为`XXX`，则该配置可以使用 `WithXXX` 进行指定，如以下代码可以指定协程池大小为 10、干运行标记为 true：
```golang
gta.StartWithOptions(db, "tasks", gta.WithPoolSize(10), gta.WithDryRun(true))
```
所有可选配置及默认值如下：
| 配置名          | 类型                                 | 默认值           | 含义                                                   |
| ------------------- | ----------------------------------------- | -------------------- | ---------------------------------------------------------- |
| Context             | context.Context                           | context.Background() | 根上下文，用于框架本身                                     |
| LoggerFactory       | func(ctx context.Context) Logger          | defaultLoggerFactory | 日志工厂方法，用于日志打印                           |
| StorageTimeout      | time.Duration                             | 1周                 | 存储超时时长，决定多久一个已完成的任务会被清理掉           |
| InitializedTimeout  | time.Duration                             | 5分钟               | 初始化超时时长，决定多久一个初始化的任务会被认定为异常 |
| RunningTimeout      | time.Duration                             | 30分钟              | 运行超时时长，决定多久一个进行中的任务会被认定为异常 |
| WaitTimeout         | time.Duration                             | 一直等待             | 等待超时时长，决定在有任务运行的情况下，` Stop` 函数最长执行多久 |
| ScanInterval        | time.Duration                             | 5秒                 | 扫描间隔时长，决定普通情况下的扫描初始化的任务的速度       |
| InstantScanInvertal | time.Duration                             | 100毫秒             | 快速扫描间隔时长，决定有未处理的初始化任务时的扫描速度     |
| CtxMarshaler        | CtxMarshaler                              | defaultCtxMarshaler  | 上下文序列化工具，决定 context 如何序列化          |
| CheckCallback       | func(logger Logger, abnormalTasks []Task) | defaultCheckCallback | 异常任务检查回调函数，决定如何处理检查到的异常任务         |
| DryRun              | bool                                      | false                | 干运行标记，用于测试，决定是否不依赖数据库干运行           |
| PoolSize            | int                                       | math.MaxInt32      | 协程池大小，底层最多用多少个协程执行任务                   |
## 单个任务定义
在调用 `Register` 进行任务注册时，需要传入对应的任务定义（TaskDefinition），具体如下：
| 配置名           | 类型                                               | 默认值        | 含义                                                     |
| -------------------- | ------------------------------------------------------ | ----------------- | ----------------------------------------------------------|
| Handler              | func(ctx context.Context, arg interface{}) (err error) | 无                | 必须，任务处理函数                                           |
| ArgType              | reflect.Type                                           | nil               | 任务入参类型，决定任务处理函数中 arg 的实际类型，如果为空，则 arg 的类型为 `map[string]interface{}` |
| CtxMarshaler         | CtxMarshaler                                           | 全局CtxMarshaler | 任务上下文序列化工具类，决定任务的 context.Context 如何序列化 |
| RetryTimes           | int                                                    | 0                 | 任务执行出错时的最大重试次数，超过该值的任务会被标记为 failed |
| RetryInterval        | func(times int) time.Duration                          | 1秒              | 任务执行出错两次重试之间的间隔                               |
| CleanSucceeded       | bool                                                   | false             | 成功后是否立即清除任务记录，若是，则任务成功后会立即清除该任务记录 |
| InitTimeoutSensitive | bool                                                   | false             | 是否对初始化超时敏感，若是，则其在初始化状态超时后不能被扫描调度 |
# 常见问题
## 什么是异常任务？如何检测异常任务？

异常任务包含超过时间还未被调度的任务、执行超时的任务，或者是因为非优雅关闭而导致的异常中止的任务

异常任务会被定时执行的内置任务检测出来并调用配置的 `CheckCallback`，默认配置下会通过日志打印异常任务的数量和对应 ID 等信息

## 协程池满了之后会阻塞吗？

目前的设计中，提交任务的步骤是不会阻塞的。协程池满以后提交的任务会让渡给其他实例去执行，同时会暂停扫描机制；当所有的实例协程池都满了以后，此时任务会积压在数据库中


## 异步任务的调度有延迟吗？

如果是基于 Commit Hook 机制则几乎没有延迟，如在协程池充足的情况下调用 `Run` 或者在内置的 `Transaction` 中调用 `RunWithTx`

如果协程池满了，或者在非内置的 `Transaction` 中调用 `RunWithTx`，则基于扫描机制调度异步任务，这时候的调度是有延迟的，延迟时间即所有实例竞争调度该任务需要的时间，和扫描间隔、协程池空闲时间、任务积压等因素有关

## 扫描机制的任务消费能力？

最大消费能力为每秒 `N * 1 / InstantScanInterval` 个任务，其中 N 为实例数量，`InstantScanInterval` 为快速扫描间隔，默认设置下单实例的消费能力为 10 个/秒。扫描机制的调度能力有限，其本身是一种辅助的调度方式，调小 `InstantScanInterval` 能够提高消费能力但同时也会提升数据库压力，故正常的情况还是尽量使用 Commit Hook 机制

## 如何处理异常和失败的任务？

在正常情况下，任务的异常和失败是小概率事件，若由于某些因素导致的任务异常和失败（如外部资源异常、异常宕机等），可以通过手动的方式进行重新调度，`TaskManager` 提供了相应的  API，如 `ForceRerunTasks` 和 `QueryUnsuccessfulTasks` 等

## 如何进行测试？

可以使用 `WithDryRun(true)` 使得框架进入干运行模式来避免其他实例读写任务表带来数据的影响，该模式下框架不会读写任务表，也不会记录任务状态等信息
## 许可证
[MIT](https://github.com/ycydsxy/gta/blob/main/LICENSE) 
