package nacos_config

import (
	bytes2 "bytes"
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/skirrund/gcloud/bootstrap/env"
	commonCfg "github.com/skirrund/gcloud/config"
)

//go:embed bootstrap.properties
var baseConfig []byte

func TestConfig(t *testing.T) {
	un := os.Environ()
	fmt.Println(un)
	opts := commonCfg.Options{
		ServerAddrs: []string{"nacos1:8848"},
		ClientOptions: commonCfg.ClientOptions{
			NamespaceId: "PBM-Service",
			LogDir:      ".",
			TimeoutMs:   5000,
			AppName:     "test-nacos",
		},
		ConfigOptions: commonCfg.ConfigOptions{
			Prefix:        "pbm-common-service",
			FileExtension: "yaml",
			Env:           "test",
			Group:         "DEFAULT_GROUP",
		},
	}
	t.Log(">>>>>")
	nacos, _ := CreateInstance(opts)
	env.GetInstance().SetBaseConfig(bytes2.NewReader(baseConfig), "properties")
	env.GetInstance().LoadProfileBaseConfig("prod", "properties")
	t.Log(nacos.GetString("datasource.dsn"))
	p := env.GetInstance().GetString("server.name")
	t.Log(">>>>>>>>>:" + p)
	t.Log("end")
	var bytes []byte
	os.Stdin.Read(bytes)
}
