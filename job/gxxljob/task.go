package gxxljob

import (
	"context"
	"fmt"
	"log/slog"
)

// TaskFunc 任务执行函数
type TaskFunc func(cxt context.Context, param *RunRequest) error

// Task 任务
type Task struct {
	Id        int64
	Name      string
	Ctx       context.Context
	Param     *RunRequest
	fn        TaskFunc
	Cancel    context.CancelFunc
	StartTime int64
	EndTime   int64
	//日志
	log *slog.Logger
}

// Run 运行任务
func (t *Task) Run(callback func(code int64, msg string)) {
	defer func(cancel func()) {
		if err := recover(); err != nil {
			t.log.Info(t.Info()+" panic: %v", err)
			callback(FailureCode, fmt.Sprintf("task panic:%v", err))
			cancel()
		}
	}(t.Cancel)
	err := t.fn(t.Ctx, t.Param)
	if err != nil {
		callback(FailureCode, err.Error())
	} else {
		callback(SuccessCode, "")
	}

}

// Info 任务信息
func (t *Task) Info() string {
	return fmt.Sprintf("任务ID[%d]任务名称[%s]参数:%s", t.Id, t.Name, t.Param.ExecutorParams)
}
