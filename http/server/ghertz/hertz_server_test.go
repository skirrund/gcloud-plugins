package ghertz

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
	nacos_registry "github.com/skirrund/gcloud-plugins/nacos/registry"
	"github.com/skirrund/gcloud/bootstrap"
	"github.com/skirrund/gcloud/registry"
	"github.com/skirrund/gcloud/server"
	"github.com/skirrund/gcloud/server/feign"
)

type Test struct {
	Id   int64
	Id2  int64
	Code string
}

func TestHertzServer(t *testing.T) {
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
	reg := nacos_registry.NewRegistry(ops)
	options := server.Options{
		ServerName: "hertz_test",
		Address:    ":8080",
	}
	srv := NewServer(options, func(engine *hertzServer.Hertz) {
		engine.GET("/test", func(c context.Context, ctx *app.RequestContext) {
			app := bootstrap.Application{
				Registry: reg,
			}
			bootstrap.MthApplication = &app
			client := feign.Client{
				ServiceName: "pbm-common-wechat-service",
			}
			client.Get("/test", nil, nil, nil)
			//i1, err = reg.SelectInstances("pbm-common-wechat-service")
			ctx.JSON(200, nil)
		})
		engine.GET("/del", func(c context.Context, ctx *app.RequestContext) {
			ClearCookie(ctx, "", "", "test")
			//t := &Test{}
			// SetCookie()
			//ctx.JSON(http.StatusOK, s)
		})
	})
	srv.Run(func() {
		fmt.Println("shut down")
	})
}
