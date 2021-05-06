# GTA - Go Task Async

一个轻量的可靠异步任务和事务消息框架

[![Go Report Card](https://goreportcard.com/badge/github.com/ycydsxy/gta)](https://goreportcard.com/report/github.com/ycydsxy/gta)
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/ycydsxy/gta/Go?logo=github)](https://github.com/ycydsxy/gta/actions/workflows/go.yml)
![Travis (.com)](https://img.shields.io/travis/com/ycydsxy/gta?label=test&logo=travis)
[![Coverage](https://img.shields.io/codecov/c/github/ycydsxy/gta?logo=codecov)](https://codecov.io/gh/ycydsxy/gta)
[![GitHub issues](https://img.shields.io/github/issues/ycydsxy/gta)](https://github.com/ycydsxy/gta/issues)
[![Release](https://img.shields.io/github/v/release/ycydsxy/gta.svg)](https://github.com/ycydsxy/gta/releases)
[![GitHub license](https://img.shields.io/github/license/ycydsxy/gta)](https://github.com/ycydsxy/gta/blob/main/LICENSE)

## 简介
GTA(Go Task Async) 是一个 Golang 实现的轻量可靠异步任务和事务消息框架，该框架有如下一些特性：
- 高可靠性：保证异步任务 At Least Once 级别的调度和执行，所有提交的任务状态可追溯
- 灵活的配置：提供了多个简单易用的可选配置项，能够较好地贴合不同场景的需求
- 允许提交多个任务：允许在同一个事务中提交多个任务，不保证任务按提交的顺序执行
- 允许提交嵌套任务：允许在提交的任务中提交新的异步任务，保证任务按提交的顺序执行
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

## 许可证
[MIT](https://github.com/ycydsxy/gta/blob/main/LICENSE) 
