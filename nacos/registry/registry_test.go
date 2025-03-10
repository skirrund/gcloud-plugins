package nacos_registry

import (
	"fmt"
	"testing"

	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/registry"
	"github.com/skirrund/gcloud/utils"
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
	reg := NewRegistry(ops)
	i := reg.GetInstance("pbm-common-wechat-service")
	i1, err := reg.SelectInstances("pbm-common-wechat-service")
	str, _ := utils.MarshalToString(i1)
	fmt.Println(str, err)
	logger.Info(i1)
	logger.Info(i)
}
