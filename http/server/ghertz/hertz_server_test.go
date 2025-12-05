package ghertz

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/gin-gonic/gin"
	"github.com/skirrund/gcloud-plugins/http/client/hertz"
	nacos_registry "github.com/skirrund/gcloud-plugins/nacos/registry"
	"github.com/skirrund/gcloud/bootstrap"
	"github.com/skirrund/gcloud/logger"
	gGin "github.com/skirrund/gcloud/plugins/server/http/gin"
	"github.com/skirrund/gcloud/registry"
	"github.com/skirrund/gcloud/server"
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
			ServiceName: "nacos_reg_test",
			ServicePort: 8081,
			Version:     "0.1",
		},
	}
	reg := nacos_registry.NewRegistry(ops)
	options := server.Options{
		ServerName: "nacos_reg_test",
		Address:    ":8899",
		H2C:        true,
	}
	reg.RegisterInstance()
	reg.Subscribe("nacos_reg_test")
	gApp := bootstrap.Application{
		Registry: reg,
	}
	bootstrap.MthApplication = &gApp
	// lb.GetInstance().SetHttpClient(hertz.GetDefaultClient())
	srv := NewServer(options, func(engine *hertzServer.Hertz) {
		engine.GET("/test", func(c context.Context, ctx *app.RequestContext) {
			ctx.JSON(200, "Get")
		})
		engine.POST("/test", func(c context.Context, ctx *app.RequestContext) {
			// mfh, err := ctx.FormFile("file")
			// fmt.Println(err)
			// mf, err := mfh.Open()
			// fmt.Println(err)
			// defer mf.Close()
			// bytes, _ := io.ReadAll(mf)
			// os.WriteFile("/Users/jerry.shi/Desktop/"+mfh.Filename, bytes, os.ModePerm)
			// hn := ctx.Host()
			// ck := cookie.Cookie{
			// 	Key:      "test",
			// 	Value:    "testV=",
			// 	Domain:   string(hn),
			// 	Path:     "/",
			// 	MaxAge:   1000000,
			// 	Secure:   false,
			// 	HttpOnly: false,
			// 	SameSite: cookie.CookieSameSiteNoneMode,
			// }
			// SetCookie(ck, ctx)
			// ck1 := cookie.Cookie{
			// 	Key:      "test2",
			// 	Value:    "testV2==",
			// 	Domain:   string(hn),
			// 	Path:     "/",
			// 	MaxAge:   1000000,
			// 	Secure:   false,
			// 	HttpOnly: false,
			// 	SameSite: cookie.CookieSameSiteNoneMode,
			// }
			// SetCookie(ck1, ctx)
			tc := GetTraceContext(ctx)
			DemoTrace(tc)
			ctx.JSON(200, "ok-hertz")
		})
		engine.GET("/del", func(c context.Context, ctx *app.RequestContext) {
			ClearCookie(ctx, "", "", "test")
			//t := &Test{}
			// SetCookie()
			//ctx.JSON(http.StatusOK, s)
		})
		engine.Any("/test1", func(c context.Context, ctx *app.RequestContext) {
			hertz.GetDefaultClient().ProxyService("nacos_reg_test", "/test", ctx, 0)
		})
	})
	srv.Run(func() {
		fmt.Println("shut down")
	})
}

func DemoTrace(ctx context.Context) {
	logger.InfoContext(ctx, "trace...")
}

// func TestHertzServer1(t *testing.T) {
// 	N201 := "nacos1:8848"
// 	ops := registry.Options{
// 		ServerAddrs: []string{N201},
// 		ClientOptions: registry.ClientOptions{
// 			AppName:             "test",
// 			LogDir:              "/Users/jerry.shi/logs/nacos/go",
// 			NotLoadCacheAtStart: true,
// 		},
// 		RegistryOptions: registry.RegistryOptions{
// 			ServiceName: "hertz_test",
// 			ServicePort: 8081,
// 			Version:     "0.1",
// 		},
// 	}
// 	reg := nacos_registry.NewRegistry(ops)
// 	options := server.Options{
// 		ServerName: "hertz_test",
// 		Address:    ":8081",
// 	}
// 	reg.RegisterInstance()
// 	reg.Subscribe("hertz_test")
// 	gApp := bootstrap.Application{
// 		Registry: reg,
// 	}
// 	bootstrap.MthApplication = &gApp
// 	srv := NewServer(options, func(engine *hertzServer.Hertz) {
// 		engine.GET("/test", func(c context.Context, ctx *app.RequestContext) {
// 			// client := feign.Client{
// 			// 	ServiceName: "pbm-common-wechat-service",
// 			// }
// 			// client.Get("/test", nil, nil, nil)

// 			is, _ := reg.SelectInstances("hertz_test")
// 			ctx.JSON(200, is)
// 		})
// 		engine.GET("/del", func(c context.Context, ctx *app.RequestContext) {
// 			ClearCookie(ctx, "", "", "test")
// 			//t := &Test{}
// 			// SetCookie()
// 			//ctx.JSON(http.StatusOK, s)
// 		})
// 		engine.GET("/test1", func(c context.Context, ctx *app.RequestContext) {
// 			hertz.GetDefaultClient().ProxyService("hertz_test", "/test", ctx, 0)
// 		})
// 	})
// 	srv.Run(func() {
// 		fmt.Println("shut down")
// 	})
// }

func TestGinServer(t *testing.T) {
	N201 := "nacos1:8848"
	ops := registry.Options{
		ServerAddrs: []string{N201},
		ClientOptions: registry.ClientOptions{
			AppName:             "test",
			LogDir:              "/Users/jerry.shi/logs/nacos/go",
			NotLoadCacheAtStart: true,
		},
		RegistryOptions: registry.RegistryOptions{
			ServiceName: "nacos_reg_test",
			ServicePort: 8899,
			Version:     "0.1",
		},
	}
	reg := nacos_registry.NewRegistry(ops)
	options := server.Options{
		ServerName: "nacos_reg_test",
		Address:    ":8899",
		H2C:        true,
	}
	reg.RegisterInstance()
	reg.Subscribe("nacos_reg_test")
	gApp := bootstrap.Application{
		Registry: reg,
	}
	bootstrap.MthApplication = &gApp
	// lb.GetInstance().SetHttpClient(hertz.GetDefaultClient())
	srv := gGin.NewServer(options, func(engine *gin.Engine) {
		engine.GET("/test", func(ctx *gin.Context) {
			ctx.JSON(200, "Get")
		})
		engine.POST("/upload", func(ctx *gin.Context) {
			if fh, err := ctx.FormFile("file"); err == nil {
				f, err := fh.Open()
				if err != nil {
					ctx.JSON(200, err.Error())
					return
				}
				fb, _ := io.ReadAll(f)
				os.WriteFile("/Users/jerry.shi/Desktop/test_"+fh.Filename, fb, os.ModePerm)
				ctx.JSON(200, "success")
			} else {
				ctx.JSON(200, err.Error())
			}
		})
		engine.POST("/test", func(ctx *gin.Context) {
			// mfh, err := ctx.FormFile("file")
			// fmt.Println(err)
			// mf, err := mfh.Open()
			// fmt.Println(err)
			// defer mf.Close()
			// bytes, _ := io.ReadAll(mf)
			// os.WriteFile("/Users/jerry.shi/Desktop/"+mfh.Filename, bytes, os.ModePerm)
			// hn := ctx.Host()
			// ck := cookie.Cookie{
			// 	Key:      "test",
			// 	Value:    "testV=",
			// 	Domain:   string(hn),
			// 	Path:     "/",
			// 	MaxAge:   1000000,
			// 	Secure:   false,
			// 	HttpOnly: false,
			// 	SameSite: cookie.CookieSameSiteNoneMode,
			// }
			// SetCookie(ck, ctx)
			// ck1 := cookie.Cookie{
			// 	Key:      "test2",
			// 	Value:    "testV2==",
			// 	Domain:   string(hn),
			// 	Path:     "/",
			// 	MaxAge:   1000000,
			// 	Secure:   false,
			// 	HttpOnly: false,
			// 	SameSite: cookie.CookieSameSiteNoneMode,
			// }
			// SetCookie(ck1, ctx)
			ctx.JSON(200, "ok-hertz")
		})
		engine.Any("/test1", func(ctx *gin.Context) {
			gGin.ProxyService("nacos_reg_test", "/upload", ctx, 0)
		})
	})
	srv.Run(func() {
		fmt.Println("shut down")
	})
}
