package nacos_config

import (
	"fmt"
	"io"
	"os"
	"strings"

	nacos "github.com/skirrund/gcloud-plugins/nacos"
	"github.com/skirrund/gcloud/bootstrap/env"
	commonConfig "github.com/skirrund/gcloud/config"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/parser"

	"github.com/skirrund/gcloud/server"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
)

const (
	NACOS_CONFIG_PREFIX_KEY          = "nacos.config.prefix"
	NACOS_CONFIG_FILE_EXTENSION_KEY  = "nacos.config.file-extension"
	NACOS_CONFIG_SERVER_ADDR_KEY     = "nacos.config.server-addr"
	NACOS_CONFIG_GROUP_KEY           = "nacos.config.group"
	NACOS_CONFIG_NAMESPACE_KEY       = "nacos.config.namespace"
	NACOS_CONFIG_NotLoadCacheAtStart = "nacos.config.notLoadCacheAtStart"
)

type nacosConfigCenter struct {
	opts   commonConfig.Options
	client config_client.IConfigClient
}

func (nc *nacosConfigCenter) LoadProfileBaseConfig(profile string, configType string) {
}

var config *viper.Viper
var nc *nacosConfigCenter

func configure(n *nacosConfigCenter, opts commonConfig.Options) error {
	client, err := nacos.CreateConfigClient(opts)
	if err != nil {
		return err
	}
	n.client = client
	return nil
}

func defaultOptions() commonConfig.Options {
	cfg := env.GetInstance()
	addr := cfg.GetString(NACOS_CONFIG_SERVER_ADDR_KEY)
	fe := cfg.GetString(NACOS_CONFIG_FILE_EXTENSION_KEY)
	group := cfg.GetString(NACOS_CONFIG_GROUP_KEY)
	ns := cfg.GetString(NACOS_CONFIG_NAMESPACE_KEY)
	prefix := cfg.GetString(NACOS_CONFIG_PREFIX_KEY)
	host, _ := os.Hostname()
	serverName := cfg.GetString(env.SERVER_SERVERNAME_KEY)
	profile := cfg.GetString(env.SERVER_PROFILE_KEY)
	dir := cfg.GetString(env.LOGGER_DIR_KEY) + "/" + serverName + "/" + host
	logger.Info("[Bootstrap] start init nacos config center properties:[addrs=" + addr + "]" + ",[FileExtension=" + fe + "],[Group=" + group + "],[Prefix=" + prefix + "],[Namespace=" + ns + "],[Env=" + profile + "]")
	notLoadCacheAtStartStr := cfg.GetString(NACOS_CONFIG_NotLoadCacheAtStart)
	notLoadCacheAtStart := true
	if strings.EqualFold(notLoadCacheAtStartStr, "false") {
		notLoadCacheAtStart = false
	}
	options := commonConfig.Options{
		ServerAddrs: strings.Split(addr, ","),
		ClientOptions: commonConfig.ClientOptions{
			NamespaceId: ns,
			LogDir:      dir,
			//CacheDir:    dir,
			TimeoutMs:           5000,
			AppName:             serverName,
			NotLoadCacheAtStart: notLoadCacheAtStart,
		},
		ConfigOptions: commonConfig.ConfigOptions{
			Prefix:        prefix,
			FileExtension: fe,
			Env:           profile,
			Group:         group,
		},
	}
	return options
}

func CreateDefaultInstance() (commonConfig.IConfig, error) {
	return CreateInstance(defaultOptions())
}

func CreateInstance(opts commonConfig.Options) (commonConfig.IConfig, error) {
	nc = &nacosConfigCenter{}
	nc.opts = opts
	err := configure(nc, opts)
	if err != nil {
		logger.Panic("[nacos]config error:", err.Error())
		return nc, err
	}
	config = parser.NewDefaultParser()
	err = nc.Read()
	if err != nil {
		logger.Panic(err)
		return nc, err
	}
	logger.Info("[nacos] CreateInstance EmitEvent ConfigChangeEvent")
	server.EmitEvent(server.ConfigChangeEvent, config)
	err = nc.Watch()
	return nc, err
}

func (nc *nacosConfigCenter) Set(key string, value any) {
	config.Set(key, value)
}

func (nc *nacosConfigCenter) Get(key string) any {
	return config.Get(key)
}

func (nc *nacosConfigCenter) GetStringWithDefault(key string, defaultString string) string {
	v := nc.GetString(key)
	if len(v) == 0 {
		return defaultString
	}
	return v
}

func (nc *nacosConfigCenter) GetInt(key string) int {
	return config.GetInt(key)
}

func (nc *nacosConfigCenter) GetIntWithDefault(key string, defaultInt int) int {
	v := nc.GetInt(key)
	if v == 0 {
		return defaultInt
	}
	return v
}

func (nc *nacosConfigCenter) GetInt64WithDefault(key string, defaultInt64 int64) int64 {
	v := nc.GetInt64(key)
	if v == 0 {
		return defaultInt64
	}
	return v
}
func (nc *nacosConfigCenter) GetInt64(key string) int64 {
	return config.GetInt64(key)
}
func (nc *nacosConfigCenter) GetString(key string) string {
	return config.GetString(key)
}
func (nc *nacosConfigCenter) GetStringSlice(key string) []string {
	return config.GetStringSlice(key)
}

func (nc *nacosConfigCenter) GetUint64(key string) uint64 {
	return config.GetUint64(key)
}
func (nc *nacosConfigCenter) GetUint64WithDefault(key string, defaultUint64 uint64) uint64 {
	v := nc.GetUint64(key)
	if v == 0 {
		return defaultUint64
	}
	return v
}
func (nc *nacosConfigCenter) GetUint(key string) uint {
	return config.GetUint(key)
}
func (nc *nacosConfigCenter) GetUintWithDefault(key string, defaultUint uint) uint {
	v := nc.GetUint(key)
	if v == 0 {
		return defaultUint
	}
	return v
}
func (nc *nacosConfigCenter) GetBool(key string) bool {
	return config.GetBool(key)
}
func (nc *nacosConfigCenter) GetFloat64(key string) float64 {
	return config.GetFloat64(key)
}

func (nc *nacosConfigCenter) GetStringMapString(key string) map[string]string {
	return config.GetStringMapString(key)
}

func (nc *nacosConfigCenter) MergeConfig(eventType server.EventName, eventInfo any) error {
	return nil
}

func (nc *nacosConfigCenter) SetBaseConfig(reader io.Reader, configType string) error {
	baseCfg := parser.NewDefaultParser()
	baseCfg.SetConfigName("base")
	baseCfg.SetConfigType(configType)
	err := baseCfg.ReadConfig(reader)
	if err != nil {
		return err
	}
	return config.MergeConfigMap(baseCfg.AllSettings())
}

func (c *nacosConfigCenter) Read() error {
	dataId := c.dataId()
	content, err := c.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  c.opts.ConfigOptions.Group,
	})
	if err != nil {
		return fmt.Errorf("[nacos] error reading data from nacos: %s【%v】", dataId, err)
	}
	if len(content) == 0 {
		return fmt.Errorf("[nacos] error reading data from nacos: %s【文件内容为空】", dataId)
	}
	reader := strings.NewReader(content)
	config.SetConfigType(c.opts.ConfigOptions.FileExtension)
	return config.ReadConfig(reader)
}

func (c *nacosConfigCenter) String() string {
	return "nacos"
}

func (c *nacosConfigCenter) Watch() error {
	return newConfigWatcher(c)
}

func (nc *nacosConfigCenter) dataId() string {
	opts := nc.opts.ConfigOptions
	return opts.Prefix + "-" + opts.Env + "." + opts.FileExtension
}

func newConfigWatcher(nc *nacosConfigCenter) error {
	logger.Info("[nacos] ListenConfig DataId:" + nc.dataId() + ",Group:" + nc.opts.ConfigOptions.Group)

	err := nc.client.ListenConfig(vo.ConfigParam{
		DataId: nc.dataId(),
		Group:  nc.opts.ConfigOptions.Group,
		OnChange: func(namespace, group, dataId, data string) {
			logger.Info("[nacos] config changed dataId:" + dataId + ",ns:" + namespace + ",group:" + group)
			reader := strings.NewReader(data)
			v := parser.NewDefaultParser()
			v.SetConfigType(nc.opts.ConfigOptions.FileExtension)
			err := v.ReadConfig(reader)
			if err != nil {
				logger.Error("[nacos] watch error:", err.Error())
				return
			}
			config = v
			logger.Info("[nacos] watch EmitEvent ConfigChangeEvent")
			server.EmitEvent(server.ConfigChangeEvent, v)
		},
	})
	if err != nil {
		logger.Error("[nacos] newConfigWatcher error:" + err.Error())
	}
	return err
}

func (nc *nacosConfigCenter) Shutdown() error {
	p := vo.ConfigParam{
		DataId: nc.dataId(),
		Group:  nc.opts.ConfigOptions.Group,
	}
	logger.Info("[nacos] CancelListenConfig:[dataId=" + p.DataId + "],[group=" + p.Group + "]")

	err := nc.client.CancelListenConfig(p)
	if err != nil {
		logger.Error("[nacos] CancelListenConfig error: ", err.Error())
	}
	return err
}
