package gxxljob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/skirrund/gcloud/bootstrap/env"
	gLogger "github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/utils"
)

const (
	regPath              = "/api/registry"
	regRemovePath        = "/api/registryRemove"
	callBackPath         = "/api/callback"
	AccessTokenHeaderKey = "XXL-JOB-ACCESS-TOKEN"
	DefaultBeatInterval  = 20
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Executor 执行器
type Executor struct {
	mu      sync.Mutex
	opts    Options
	address string
	regList *taskList //注册任务列表
	runList *taskList //正在执行任务列表
	logger  *slog.Logger
	//	// Init 初始化
	//	Init(...Options)
	//	// LogHandler 日志查询
	//	//LogHandler(handler LogHandler)
	//	// RegTask 注册任务
	//	//RegTask(pattern string, task TaskFunc)
	//	// RunTask 运行任务
	//	RunTask(writer http.ResponseWriter, request *http.Request)
	//	// KillTask 杀死任务
	//	KillTask(writer http.ResponseWriter, request *http.Request)
	//	// TaskLog 任务日志
	//	TaskLog(writer http.ResponseWriter, request *http.Request)
	//	// Beat 心跳检测
	//	Beat(writer http.ResponseWriter, request *http.Request)
	//	// IdleBeat 忙碌检测
	//	IdleBeat(writer http.ResponseWriter, request *http.Request)
	//	// Run 运行服务
	//	Run() error
	//	// Stop 停止服务
	//	Stop()
}

func Init(opts Options) *Executor {
	e := &Executor{}
	adminAddress := opts.AdminAddresses
	if len(adminAddress) > 0 {
		opts.adminAddresseList = strings.Split(adminAddress, ",")
	}
	ip := opts.ExecutorAddress
	if len(ip) == 0 {
		ip = utils.LocalIP()
	}
	port := opts.ExecutorPort
	if port <= 0 {
		port = DefaultExecutorPort
	}
	portStr := strconv.FormatInt(port, 10)
	e.address = ip + ":" + portStr
	appName := opts.AppName
	if len(appName) == 0 {
		appName = DefaultAppName
	}
	logger := opts.Logger
	if len(opts.LogPath) == 0 {
		opts.LogPath = "."
	}
	if logger == nil {
		logger = gLogger.NewLogInstance(opts.LogPath, opts.AppName, portStr, true, opts.LogJsonFormat, opts.Logretentiondays)
	}
	e.logger = logger
	e.opts = opts
	e.regList = &taskList{
		data: make(map[string]*Task),
	}
	e.runList = &taskList{
		data: make(map[string]*Task),
	}
	if opts.BeatInterval <= 0 {
		opts.BeatInterval = DefaultBeatInterval
	}
	go e.registry()
	return e
}

func (e *Executor) Stop() {
	e.registryRemove()
}

func RunWithDefaultOptionsLogger(logger *slog.Logger) (executor *Executor, err error) {
	return RunWithDefaultOptionsLoggerAndAppName("", slog.Default())
}

func RunWithDefaultOptionsLoggerAndAppName(appName string, logger *slog.Logger) (executor *Executor, err error) {
	opts := Options{}
	opts.Logger = logger
	utils.NewOptions(env.GetInstance(), &opts)
	if len(appName) > 0 {
		opts.WithAppName(appName)
	}
	return RunWithOptions(opts)
}

func RunWithDefaultOptions(appName string) (executor *Executor, err error) {
	return RunWithDefaultOptionsLoggerAndAppName(appName, slog.Default())
}

func RunWithOptions(opts Options) (executor *Executor, err error) {
	executor = Init(opts)
	return executor, executor.Run()
}

func (e *Executor) Run() (err error) {
	// 创建路由器
	mux := http.NewServeMux()
	// 设置路由规则
	mux.HandleFunc("/run", e.runTask)
	mux.HandleFunc("/kill", e.killTask)
	mux.HandleFunc("/log", e.taskLog)
	mux.HandleFunc("/beat", e.beat)
	mux.HandleFunc("/idleBeat", e.idleBeat)
	go func(e *Executor) {
		// 创建服务器
		server := &http.Server{
			Addr:         e.address,
			WriteTimeout: time.Second * 3,
			Handler:      mux,
		}
		// 监听端口并提供服务
		e.logger.Info("[xxljob] Starting server at " + e.address)
		err := server.ListenAndServe()
		if err != nil {
			e.logger.Error(err.Error())
			panic(err)
		}
	}(e)
	go func(e *Executor) {
		quit := make(chan os.Signal, 2)
		signal.Notify(quit, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit
		e.logger.Info("[xxljob] Received signal:" + sig.String())
		e.Stop()
	}(e)
	return nil
}

// RegTask 注册任务
func (e *Executor) RegTask(pattern string, task TaskFunc) {
	var t = &Task{}
	t.fn = task
	e.regList.Set(pattern, t)
}

// 删除一个任务
func (e *Executor) killTask(writer http.ResponseWriter, request *http.Request) {
	req, _ := io.ReadAll(request.Body)
	param := &KillRequest{}
	err := utils.Unmarshal(req, param)
	if err != nil {
		_, _ = writer.Write(commonFailWithMsgResp("killTask error:" + err.Error()))
		e.logger.Error("参数解析错误:" + string(req))
		return
	}
	jobId := param.JobID
	jobIdStr := strconv.FormatInt(jobId, 10)
	if !e.runList.Exists(jobIdStr) {
		_, _ = writer.Write(commonFailWithMsgResp("killTask error:任务不在运行中"))
		e.logger.Error("任务[" + jobIdStr + "]没有运行")
		return
	}
	task := e.runList.Get(jobIdStr)
	task.Cancel()
	e.runList.Del(jobIdStr)
	_, _ = writer.Write(commonSuccessResp())
}

// 任务日志
func (e *Executor) taskLog(writer http.ResponseWriter, request *http.Request) {
	data, _ := io.ReadAll(request.Body)
	req := &RunLogRequest{}
	err := utils.Unmarshal(data, req)
	if err != nil {
		e.logger.Error("日志请求解析失败:" + err.Error())
		_, _ = writer.Write(commonFailWithMsgResp("taskLog error:" + err.Error()))
		e.logger.Error("参数解析错误:" + string(data))
		return
	}
	res := RunLogRespContent{
		FromLineNum: req.FromLineNum,
		ToLineNum:   2,
		LogContent:  "",
		IsEnd:       true,
	}
	e.logger.Debug("日志请求参数:", slog.Attr{Key: "req", Value: slog.AnyValue(req)})
	str, _ := utils.Marshal(res)
	_, _ = writer.Write(str)
}

// 心跳检测
func (e *Executor) beat(writer http.ResponseWriter, request *http.Request) {
	e.logger.Debug("[xxljob]心跳检测")
	_, _ = writer.Write(commonSuccessResp())
}

// 忙碌检测
func (e *Executor) idleBeat(writer http.ResponseWriter, request *http.Request) {
	req, _ := io.ReadAll(request.Body)
	e.logger.Debug("[xxljob]忙碌检测>>>>" + string(req))
	param := &IdleBeatReq{}
	err := utils.Unmarshal(req, &param)
	if err != nil {
		_, _ = writer.Write(commonFailWithMsgResp("idleBeat error:" + err.Error()))
		e.logger.Error("参数解析错误:" + string(req))
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	jobIdStr := strconv.FormatInt(param.JobId, 10)
	if e.runList.Exists(jobIdStr) {
		_, _ = writer.Write(commonFailWithMsgResp("正在运行"))
		e.logger.Error("idleBeat任务[" + jobIdStr + "]正在运行")
		return
	}
	e.logger.Debug("忙碌检测任务参数:", slog.Attr{Key: "params", Value: slog.AnyValue(param)})
	_, _ = writer.Write(commonSuccessResp())
}

// 回调任务列表
func (e *Executor) callback(task *Task, code int64, msg string) {
	taskId := strconv.FormatInt(task.Id, 10)
	e.runList.Del(taskId)
	req := &JobHandleResult{
		LogID:      task.Param.LogID,
		LogDateTim: task.Param.LogDateTime,
		HandleCode: code,
		HandleMsg:  msg,
	}
	for _, addr := range e.opts.adminAddresseList {
		result, err := e.post(addr, callBackPath, []*JobHandleResult{req})
		if err != nil {
			e.logger.Error("回调任务失败:"+err.Error(), ",", result.Code, ",", result.Msg)
			return
		}
		e.logger.Debug("回调任务成功:", strconv.FormatInt(result.Code, 10), "[", result.Msg)
	}
}

// 运行一个任务
func (e *Executor) runTask(writer http.ResponseWriter, request *http.Request) {
	req, _ := io.ReadAll(request.Body)
	e.logger.Debug("[xxljob]runTask>>>>>" + string(req))
	param := &RunRequest{}
	err := utils.Unmarshal(req, param)
	if err != nil {
		_, _ = writer.Write(commonFailWithMsgResp("参数错误"))
		e.logger.Error("参数解析错误:" + string(req))
		return
	}
	e.logger.Debug("任务参数:", slog.Attr{Key: "params", Value: slog.AnyValue(param)})
	jodIdStr := strconv.FormatInt(param.JobID, 10)
	if !e.regList.Exists(param.ExecutorHandler) {
		_, _ = writer.Write(commonFailWithMsgResp("Task not registered"))
		e.logger.Error("任务["+jodIdStr, "]没有注册:", param.ExecutorHandler)
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	//阻塞策略处理
	if e.runList.Exists(jodIdStr) {
		if param.ExecutorBlockStrategy == coverEarly { //覆盖之前调度
			oldTask := e.runList.Get(jodIdStr)
			if oldTask != nil {
				oldTask.Cancel()
				e.runList.Del(jodIdStr)
			}
		} else { //单机串行,丢弃后续调度 都进行阻塞
			_, _ = writer.Write(commonFailWithMsgResp("There are tasks running"))
			e.logger.Error("任务[" + jodIdStr + "]已经在运行了:" + param.ExecutorHandler)
			return
		}
	}
	cxt := context.Background()
	task := e.regList.Get(param.ExecutorHandler)
	if param.ExecutorTimeout > 0 {
		task.Ctx, task.Cancel = context.WithTimeout(cxt, time.Duration(param.ExecutorTimeout)*time.Second)
	} else {
		task.Ctx, task.Cancel = context.WithCancel(cxt)
	}
	task.Id = param.JobID
	task.Name = param.ExecutorHandler
	task.Param = param
	task.log = e.logger
	e.runList.Set(jodIdStr, task)
	go task.Run(func(code int64, msg string) {
		e.callback(task, code, msg)
	})
	e.logger.Debug("任务[" + jodIdStr + "]开始执行:" + param.ExecutorHandler)
	_, _ = writer.Write(commonSuccessResp())
}

// 执行器注册摘除
func (e *Executor) registryRemove() {
	req := &Registry{
		RegistryGroup: DefaultRegistryGroup,
		RegistryKey:   e.opts.AppName,
		RegistryValue: DefaultRegisterAddressHttp + e.address,
	}
	if !strings.HasSuffix(req.RegistryValue, "/") {
		req.RegistryValue += "/"
	}
	e.logger.Info("[xxljob] 执行器摘除:"+DefaultRegistryGroup, "[", req.RegistryKey, " ]", req.RegistryValue)
	for _, addr := range e.opts.adminAddresseList {
		go func(url string) {
			result, err := e.post(url, regRemovePath, req)
			if err != nil {
				e.logger.Error("[xxljob] 执行器摘除失败:"+err.Error(), ",", result.Code, ",", result.Msg)
				return
			}
			e.logger.Info("[xxljob] 执行器摘除成功:", strconv.FormatInt(result.Code, 10), "[", result.Msg)
		}(addr)
	}
}

// 注册执行器到调度中心
func (e *Executor) registry() {
	beatInterval := e.opts.BeatInterval
	if beatInterval <= 0 {
		beatInterval = DefaultBeatInterval
	}
	dura := time.Second * time.Duration(beatInterval)
	t := time.NewTimer(time.Second * 0) //初始立即执行
	defer t.Stop()
	req := &Registry{
		RegistryGroup: DefaultRegistryGroup,
		RegistryKey:   e.opts.AppName,
		RegistryValue: DefaultRegisterAddressHttp + e.address,
	}
	if !strings.HasSuffix(req.RegistryValue, "/") {
		req.RegistryValue += "/"
	}
	for {
		<-t.C
		t.Reset(dura) //20秒心跳防止过期
		for _, addr := range e.opts.adminAddresseList {
			go func(url string) {
				result, err := e.post(url, regPath, req)
				if err != nil {
					e.logger.Error("执行器注册失败:"+err.Error(), ",", result.Code, ",", result.Msg)
					return
				}
				e.logger.Debug("执行器注册成功:", slog.Attr{Key: "code", Value: slog.Int64Value(result.Code)}, slog.Attr{Key: "msg", Value: slog.AnyValue(result.Msg)})
			}(addr)
		}

	}
}

func (e *Executor) post(addr, path string, body any) (resp *Resp, err error) {
	header := make(map[string]string)
	if len(e.opts.AccessToken) > 0 {
		header[AccessTokenHeaderKey] = e.opts.AccessToken
	}
	resp = &Resp{}
	bodyBytes, err := utils.Marshal(body)
	if err != nil {
		slog.Error("[xxljob] request error:" + err.Error())
		return resp, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, addr+path, bytes.NewReader(bodyBytes))
	headers := httpReq.Header
	headers.Set("Content-Type", "application/json;charset=utf-8")
	if len(e.opts.AccessToken) > 0 {
		headers.Set("AccessTokenHeaderKey", e.opts.AccessToken)
	}
	httpResp, err := httpClient.Do(httpReq)
	//_, err = gHttp.PostJSONUrl(addr+path, header, body, resp)
	if err != nil {
		return resp, err
	}
	defer httpResp.Body.Close()
	r, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, err
	}
	err = utils.Unmarshal(r, resp)
	if err != nil {
		return resp, err
	}
	if resp.Code != SuccessCode {
		return resp, errors.New("[xxljob] 请求失败")
	}
	return
}
