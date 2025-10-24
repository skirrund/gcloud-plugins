package nacos_registry

import (
	"testing"

	"github.com/skirrund/gcloud/bootstrap/env"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/registry"
)

func TestRegistry(t *testing.T) {
	N201 := "nacos1:8848"
	ops := registry.Options{
		ServerAddrs: []string{N201},
		ClientOptions: registry.ClientOptions{
			AppName:             "test",
			LogDir:              "/Users/jerry.shi/logs/nacos/go",
			NotLoadCacheAtStart: true,
		},
		RegistryOptions: registry.RegistryOptions{
			ServiceName: "test-local",
			ServicePort: 8899,
			Version:     "0.1",
		},
	}
	env.GetInstance().Set(env.SERVER_H2C_KEY, true)
	reg := NewRegistry(ops)
	reg.RegisterInstance()
	i := reg.GetInstance("test-local")
	logger.Info(i)
}
