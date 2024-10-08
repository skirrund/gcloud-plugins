package gxxljob

// 说明：触发任务执行
type RunRequest struct {
	JobID                 int64  `json:"jobId"`                 // 任务ID
	ExecutorHandler       string `json:"executorHandler"`       // 任务标识
	ExecutorParams        string `json:"executorParams"`        // 任务参数
	ExecutorBlockStrategy string `json:"executorBlockStrategy"` // 任务阻塞策略
	ExecutorTimeout       int64  `json:"executorTimeout"`       // 任务超时时间，单位秒，大于零时生效
	LogID                 int64  `json:"logId"`                 // 本次调度日志ID
	LogDateTime           int64  `json:"logDateTime"`           // 本次调度日志时间
	GlueType              string `json:"glueType"`              // 任务模式，可选值参考 com.xxl.job.core.glue.GlueTypeEnum
	GlueSource            string `json:"glueSource"`            // GLUE脚本代码
	GlueUpdatetime        int64  `json:"glueUpdatetime"`        // GLUE脚本更新时间，用于判定脚本是否变更以及是否需要刷新
	BroadcastIndex        int64  `json:"broadcastIndex"`        // 分片参数：当前分片
	BroadcastTotal        int64  `json:"broadcastTotal"`        // 分片参数：总分片
}

// 说明：终止任务
type KillRequest struct {
	JobID int64 `json:"jobId"` // 任务ID
}

// 忙碌检测:调度中心检测指定执行器上指定任务是否忙碌（运行中）时使用
type IdleBeatReq struct {
	JobId int64 `json:"jobId"` // 任务ID
}

type RunLogRequest struct {
	LogID       int64 `json:"logId"`       // 本次调度日志ID
	LogDateTime int64 `json:"logDateTime"` // 本次调度日志时间
	FromLineNum int64 `json:"fromLineNum"` // 日志开始行号，滚动加载日志
}
