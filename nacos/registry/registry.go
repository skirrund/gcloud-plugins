package nacos_registry

import (
	"os"
	"runtime"
	"strings"

	nacos "github.com/skirrund/gcloud-plugins/nacos"
	"github.com/skirrund/gcloud/bootstrap/env"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/registry"

	"github.com/skirrund/gcloud/server"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// type Options struct {
// 	Addrs     []string
// 	Timeout   time.Duration
// 	Secure    bool
// 	TLSConfig *tls.Config
// 	// Other options for implementations of the interface
// 	// can be stored in a context
// 	Context context.Context
// }

type nacosRegistry struct {
	client naming_client.INamingClient
	opts   registry.Options
}

const (
	NACOS_DISCOVERY_SERVER_ADDE_KEY     = "nacos.discovery.server-addr"
	NACOS_DISCOVERY_NotLoadCacheAtStart = "nacos.discovery.notLoadCacheAtStart"
)

var registryCenter *nacosRegistry

func defaultOptions() registry.Options {
	cfg := env.GetInstance()
	addr := cfg.GetString(NACOS_DISCOVERY_SERVER_ADDE_KEY)
	serverName := cfg.GetString(env.SERVER_SERVERNAME_KEY)
	serverPort := cfg.GetUint64(env.SERVER_PORT_KEY)
	host, _ := os.Hostname()
	dir := cfg.GetString(env.LOGGER_DIR_KEY) + "/" + serverName + "/" + host
	notLoadCacheAtStartStr := cfg.GetString(NACOS_DISCOVERY_NotLoadCacheAtStart)
	notLoadCacheAtStart := true
	if len(notLoadCacheAtStartStr) > 0 && strings.EqualFold(notLoadCacheAtStartStr, "false") {
		notLoadCacheAtStart = false
	}
	options := registry.Options{
		ServerAddrs: strings.Split(addr, ","),
		ClientOptions: registry.ClientOptions{
			//NamespaceId: ns,
			LogDir: dir,
			//CacheDir:  dir,
			TimeoutMs:           3000,
			AppName:             serverName,
			NotLoadCacheAtStart: notLoadCacheAtStart,
		},
		RegistryOptions: registry.RegistryOptions{
			ServiceName: serverName,
			ServicePort: serverPort,
		},
	}
	return options
}

func DefaultRegistry() registry.IRegistry {
	return NewRegistry(defaultOptions())
}

func NewRegistry(opts registry.Options) registry.IRegistry {
	registryCenter = &nacosRegistry{}
	registryCenter.opts = opts
	err := configure(registryCenter, opts)
	if err != nil {
		logger.Panic("[nacos] NewRegistry err:" + err.Error())
	}
	//err := registryCenter.RegisterInstance()
	//	if err != nil {
	///	logger.Logger.Panic("[nacos] RegisterInstance err:" + err.Error())
	//	}
	return registryCenter
}

func configure(n *nacosRegistry, opts registry.Options) error {
	client, err := nacos.CreateNamingClient(opts)
	if err != nil {
		return err
	}
	n.client = client

	return nil
}

func (nr *nacosRegistry) RegisterInstance() error {
	opts := nr.opts.RegistryOptions
	registryParam := vo.RegisterInstanceParam{
		Ip:          util.LocalIP(),
		Port:        opts.ServicePort,
		ServiceName: opts.ServiceName,
		Weight:      1.0,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"version": opts.Version, "preserved.register.source": "http/go-" + runtime.Version()},
		//ClusterName: "cluster-a", // default value is DEFAULT
		GroupName: opts.Group, // default value is DEFAULT_GROUP
	}
	logger.Info("[nacos]  RegisterInstance:", registryParam)
	_, err := nr.client.RegisterInstance(registryParam)
	if err != nil {
		logger.Error("[nacos]  RegisterInstance error:", err.Error())
		panic(err)
	}
	//if !success {
	//	logger.Error("[nacos]  RegisterInstance fail")
	//} else {
	//	logger.Info("[nacos]  RegisterInstance success")
	//}
	return err
}

func (nr *nacosRegistry) Degister() error {
	opts := nr.opts.RegistryOptions
	params := vo.DeregisterInstanceParam{
		Ip:          util.LocalIP(),
		Port:        opts.ServicePort,
		ServiceName: opts.ServiceName,
		Ephemeral:   true,
		//ClusterName: "cluster-a", // default value is DEFAULT
		GroupName: opts.Group, // default value is DEFAULT_GROUP
	}
	logger.Info("[nacos] degister service:", params)
	_, err := nr.client.DeregisterInstance(params)
	if err != nil {
		logger.Error("[nacos] degister service:" + opts.ServiceName + " fail," + err.Error())
	}
	return err
}

func (nr *nacosRegistry) GetInstance(serviceName string) *registry.Instance {
	instance, err := nr.client.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   nr.opts.RegistryOptions.Group,
		// Clusters:    []string{"DEFAULT"}, // default value is DEFAULT
	})
	if err != nil {
		logger.Error("[nacos] GetInstance error:" + err.Error())
		return nil
	}
	return &registry.Instance{
		Ip:       instance.Ip,
		Port:     instance.Port,
		Metadata: instance.Metadata,
	}
}

func (nr *nacosRegistry) SelectInstances(serviceName string) ([]*registry.Instance, error) {
	instance, err := nr.client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   nr.opts.RegistryOptions.Group,
		// Clusters:    []string{"DEFAULT"}, // default value is DEFAULT
		HealthyOnly: true,
	})
	if err != nil {
		logger.Error("[nacos] GetInstance error:" + err.Error())
		return nil, err
	}
	var instances = make([]*registry.Instance, len(instance))
	for i, ins := range instance {
		instances[i] = &registry.Instance{
			Ip:       ins.Ip,
			Port:     ins.Port,
			Metadata: ins.Metadata,
		}
	}
	return instances, nil
}

func (nr *nacosRegistry) Subscribe(serviceName string) error {
	err := nr.client.Subscribe(&vo.SubscribeParam{
		ServiceName: serviceName,
		GroupName:   nr.opts.RegistryOptions.Group, // default value is DEFAULT_GROUP
		// Clusters:    []string{"DEFAULT"},           // default value is DEFAULT
		SubscribeCallback: func(services []model.Instance, err error) {
			logger.Info("[nacos] registry change:", services)
			var instances = make([]*registry.Instance, len(services))
			for i, ins := range services {
				instances[i] = &registry.Instance{
					Ip:       ins.Ip,
					Port:     ins.Port,
					Metadata: ins.Metadata,
				}
			}

			server.EmitEvent(server.RegistryChangeEvent, map[string][]*registry.Instance{serviceName: instances})
		},
	})
	return err
}

func (nr *nacosRegistry) Shutdown() {
	nr.Degister()
}
