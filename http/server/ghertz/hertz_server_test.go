package ghertz

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/skirrund/gcloud/server"
	"github.com/skirrund/gcloud/server/http/cookie"
)

type Test struct {
	Id   int64
	Id2  int64
	Code string
}

func TestHertzServer(t *testing.T) {
	options := server.Options{
		ServerName: "hertz_test",
		Address:    ":8080",
	}
	srv := NewServer(options, func(engine *hertzServer.Hertz) {
		engine.GET("/test", func(c context.Context, ctx *app.RequestContext) {
			co := cookie.Cookie{
				Key:    "test",
				Value:  "123",
				MaxAge: 1800,
			}
			SetCookie(co, ctx)
			co = cookie.Cookie{
				Key:    "test2",
				Value:  "123",
				MaxAge: 1800,
			}
			SetCookie(co, ctx)
			qStr := ctx.Request.RequestURI()
			str := string(qStr)
			fmt.Println(str)
			//t := &Test{}
			// SetCookie()
			//s := GetCookie("test", ctx)
			ctx.JSON(http.StatusOK, str)
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
