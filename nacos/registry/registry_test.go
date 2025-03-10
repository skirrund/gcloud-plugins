package nacos_registry

import (
	"testing"

	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/registry"
)

func TestRegistry(t *testing.T) {
	N201 := "nacos1:8848"
	ops := registry.Options{
		ServerAddrs: []string{N201},
		ClientOptions: registry.ClientOptions{
			AppName: "test",
			LogDir:  "/Users/jerry.shi/logs/nacos/go",
		},
		RegistryOptions: registry.RegistryOptions{
			ServiceName: "test-local",
			ServicePort: 8899,
			Version:     "0.1",
		},
	}
	reg := NewRegistry(ops)
	reg.RegisterInstance()
	i := reg.GetInstance("pbm-common-wechat-service")
	logger.Info(i)
}
