package nacos

import (
	"net"
	"os"
	"strconv"

	"github.com/skirrund/gcloud/config"
	"github.com/skirrund/gcloud/registry"

	"github.com/skirrund/gcloud/logger"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

const (
	SYSENV_NACOS_USERNAME = "NACOS_USERNAME"
	SYSENV_NACOS_PASSWORD = "NACOS_PASSWORD"
)

func createServerConfig(addrs []string, contextPath string) []constant.ServerConfig {
	serverConfigs := make([]constant.ServerConfig, 0)
	if len(contextPath) == 0 {
		contextPath = "/nacos"
	}

	// iterate the options addresses
	for _, address := range addrs {
		// check we have a port
		addr, port, err := net.SplitHostPort(address)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			serverConfigs = append(serverConfigs, constant.ServerConfig{
				IpAddr:      addr,
				Port:        8848,
				ContextPath: contextPath,
			})
		} else if err == nil {
			p, err := strconv.ParseUint(port, 10, 64)
			if err != nil {
				continue
			}
			serverConfigs = append(serverConfigs, constant.ServerConfig{
				IpAddr:      addr,
				Port:        p,
				ContextPath: contextPath,
			})
		}
	}
	return serverConfigs
}

func createClientConfig(opts any) constant.ClientConfig {
	var clientConfig constant.ClientConfig
	if ccfg, ok := opts.(config.ClientOptions); ok {
		logger.Infof("[nacos] init config client:%v", ccfg)
		clientConfig = constant.ClientConfig{
			LogLevel:            "error",
			TimeoutMs:           ccfg.TimeoutMs,
			LogDir:              ccfg.LogDir,
			CacheDir:            ccfg.CacheDir,
			NamespaceId:         ccfg.NamespaceId,
			AppName:             ccfg.AppName,
			ContextPath:         ccfg.ContextPath,
			NotLoadCacheAtStart: ccfg.NotLoadCacheAtStart,
		}
		username := ccfg.Username
		password := ccfg.Password
		if len(username) == 0 && len(password) == 0 {
			username = os.Getenv(SYSENV_NACOS_USERNAME)
			password = os.Getenv(SYSENV_NACOS_PASSWORD)
		}
		if len(username) > 0 && len(password) > 0 {
			clientConfig.Username = username
			clientConfig.Password = password
		}
	}
	if rcfg, ok := opts.(registry.ClientOptions); ok {
		logger.Infof("[nacos] init registry client:%v", rcfg)
		clientConfig = constant.ClientConfig{
			LogLevel:            "error",
			TimeoutMs:           rcfg.TimeoutMs,
			LogDir:              rcfg.LogDir,
			CacheDir:            rcfg.CacheDir,
			NamespaceId:         rcfg.NamespaceId,
			AppName:             rcfg.AppName,
			ContextPath:         rcfg.ContextPath,
			NotLoadCacheAtStart: rcfg.NotLoadCacheAtStart,
		}
		username := rcfg.Username
		password := rcfg.Password
		if len(username) == 0 && len(password) == 0 {
			username = os.Getenv(SYSENV_NACOS_USERNAME)
			password = os.Getenv(SYSENV_NACOS_PASSWORD)
		}
		if len(username) > 0 && len(password) > 0 {
			clientConfig.Username = username
			clientConfig.Password = password
		}
	}
	return clientConfig
}

func CreateConfigClient(opts config.Options) (config_client.IConfigClient, error) {
	addrs := opts.ServerAddrs
	sc := createServerConfig(addrs, opts.ClientOptions.ContextPath)
	cc := createClientConfig(opts.ClientOptions)
	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	return client, err
}

func CreateNamingClient(opts registry.Options) (naming_client.INamingClient, error) {
	addrs := opts.ServerAddrs
	sc := createServerConfig(addrs, opts.ClientOptions.ContextPath)
	cc := createClientConfig(opts.ClientOptions)
	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	return client, err
}
