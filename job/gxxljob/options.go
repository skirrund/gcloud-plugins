package gxxljob

import (
	"log/slog"
	"time"
)

type Options struct {
	// 调度中心部署根地址 [选填]：如调度中心集群部署存在多个地址则用逗号分隔。执行器将会使用该地址进行"执行器心跳注册"和"任务结果回调"；为空则关闭自动注册
	AdminAddresses    string `json:"admin_addresses" property:"xxl.job.admin.addresses"`
	adminAddresseList []string
	//执行器通讯TOKEN [选填]：非空时启用；
	AccessToken string `json:"access_token" property:"xxl.job.accessToken"`
	//执行器AppName [选填]：执行器心跳注册分组依据；为空则关闭自动注册
	AppName string        `json:"app_name" property:"xxl.job.executor.appname"`
	Timeout time.Duration `json:"timeout" property:"xxl.job.executor.timeout"` //接口超时时间
	//执行器注册 [选填]：优先使用该配置作为注册地址，为空时使用内嵌服务 ”IP:PORT“ 作为注册地址。从而更灵活的支持容器类型执行器动态IP和动态映射端口问题
	ExecutorAddress string `json:"executor_address" property:"xxl.job.executor.address"`
	//执行器端口号 [选填]：小于等于0则自动获取；默认端口为9999，单机部署多个执行器时，注意要配置不同执行器端口；
	ExecutorPort int64 `json:"executor_port" property:"xxl.job.executor.port"`
	Logger       *slog.Logger
	// 执行器运行日志文件存储磁盘路径 [选填] ：需要对该路径拥有读写权限；为空则使用默认路径；
	LogPath string `json:"log_path" property:"xxl.job.executor.logpath"`
	//执行器日志文件保存天数 [选填] ： 过期日志自动清理, 限制值大于等于3时生效; 否则, 如-1, 关闭自动清理功能；
	Logretentiondays uint64 `json:"logretentiondays" property:"xxl.job.executor.logretentiondays"`
	LogJsonFormat    bool   `json:"logJsonFormat" property:"xxl.job.executor.logJsonFormat"`
	//执行器心跳间隔单位秒
	BeatInterval uint64 `json:"beatInterval" property:"xxl.job.executor.beatInterval"`
}

func (opt *Options) WithAdminAddresses(adminAddresses string) *Options {
	opt.AdminAddresses = adminAddresses
	return opt
}
func (opt *Options) WithAccessToken(accessToken string) *Options {
	opt.AccessToken = accessToken
	return opt
}
func (opt *Options) WithAppName(appName string) *Options {
	opt.AppName = appName
	return opt
}
func (opt *Options) WithExecutorAddress(executorAddress string) *Options {
	opt.ExecutorAddress = executorAddress
	return opt
}
func (opt *Options) WithExecutorPort(executorPort int64) *Options {
	opt.ExecutorPort = executorPort
	return opt
}
func (opt *Options) WithLogPath(logPath string) *Options {
	opt.LogPath = logPath
	return opt
}
func (opt *Options) WithLogger(logger *slog.Logger) *Options {
	opt.Logger = logger
	return opt
}
func (opt *Options) WithLogretentiondays(logretentiondays uint64) *Options {
	opt.Logretentiondays = logretentiondays
	return opt
}
