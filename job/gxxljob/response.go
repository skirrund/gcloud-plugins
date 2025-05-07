package gxxljob

import "github.com/skirrund/gcloud/utils"

// 接口响应结果
type Resp struct {
	Code    int64 `json:"code"` // 200 表示正常、其他失败
	Msg     any   `json:"msg"`  // 错误提示消息
	Content any   `json:"content"`
}

type JobHandleResult struct {
	LogID      int64  `json:"logId"`      // 本次调度日志ID
	LogDateTim int64  `json:"logDateTim"` // 本次调度日志时间
	HandleCode int64  `json:"handleCode"` //200表示正常,500表示失败
	HandleMsg  string `json:"handleMsg"`
}

// LogResContent 日志响应内容
type RunLogRespContent struct {
	FromLineNum int64  `json:"fromLineNum"` // 本次请求，日志开始行数
	ToLineNum   int64  `json:"toLineNum"`   // 本次请求，日志结束行号
	LogContent  string `json:"logContent"`  // 本次请求日志内容
	IsEnd       bool   `json:"isEnd"`       // 日志是否全部加载完
}

func commonSuccessResp() []byte {
	return commonResp(SuccessCode, "")
}

func commonFailWithMsgResp(msg string) []byte {
	return commonResp(FailureCode, msg)
}

func commonResp(code int64, msg string) []byte {
	data := &Resp{
		Code: code,
		Msg:  msg,
	}
	str, _ := utils.Marshal(data)
	return str
}
