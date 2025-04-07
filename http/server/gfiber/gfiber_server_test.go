package gfiber

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/skirrund/gcloud-plugins/http/client/fasthttp"
	nacos_registry "github.com/skirrund/gcloud-plugins/nacos/registry"
	"github.com/skirrund/gcloud/bootstrap"
	"github.com/skirrund/gcloud/registry"
	"github.com/skirrund/gcloud/server"
	"github.com/skirrund/gcloud/server/http/cookie"
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
			ServiceName: "hertz_test",
			ServicePort: 8080,
			Version:     "0.1",
		},
	}
	reg := nacos_registry.NewRegistry(ops)
	options := server.Options{
		ServerName: "hertz_test",
		Address:    ":8080",
	}
	reg.RegisterInstance()

	ops1 := registry.Options{
		ServerAddrs: []string{N201},
		ClientOptions: registry.ClientOptions{
			AppName:             "test",
			LogDir:              "/Users/jerry.shi/logs/nacos/go",
			NotLoadCacheAtStart: true,
		},
		RegistryOptions: registry.RegistryOptions{
			ServiceName: "hertz_test",
			ServicePort: 8081,
			Version:     "0.1",
		},
	}
	reg1 := nacos_registry.NewRegistry(ops1)
	reg1.RegisterInstance()
	reg.Subscribe("hertz_test")
	gApp := bootstrap.Application{
		Registry: reg,
	}
	bootstrap.MthApplication = &gApp
	//lb.GetInstance().SetHttpClient(fasthttp.GetDefaultClient())
	srv := NewServer(options, func(engine *fiber.App) {
		engine.Post("/test", func(ctx *fiber.Ctx) error {
			mfh, _ := ctx.FormFile("file")
			mf, _ := mfh.Open()
			defer mf.Close()
			bytes, _ := io.ReadAll(mf)
			os.WriteFile("/Users/jerry.shi/Desktop/"+mfh.Filename, bytes, os.ModePerm)
			hn := ctx.Hostname()
			ck := cookie.Cookie{
				Key:      "test",
				Value:    "testV",
				Domain:   hn,
				Path:     "/",
				MaxAge:   100000,
				Secure:   false,
				HttpOnly: false,
				SameSite: cookie.CookieSameSiteNoneMode,
			}
			SetCookie(ck, ctx)
			ck1 := cookie.Cookie{
				Key:      "test2",
				Value:    "testV2",
				Domain:   hn,
				Path:     "/",
				MaxAge:   100000,
				Secure:   false,
				HttpOnly: false,
				SameSite: cookie.CookieSameSiteNoneMode,
			}
			SetCookie(ck1, ctx)
			return ctx.JSON("ok")

		})
		engine.Get("/del", func(ctx *fiber.Ctx) error {
			ClearCookie(ctx, "", "", "test")
			//t := &Test{}
			// SetCookie()
			//ctx.JSON(http.StatusOK, s)
			return nil
		})
		engine.Post("/test1", func(ctx *fiber.Ctx) error {
			return fasthttp.GetDefaultClient().ProxyService("hertz_test", "/test", ctx, 5*time.Minute)
		})
	})
	srv.Run(func() {
		fmt.Println("shut down")
	})
}
