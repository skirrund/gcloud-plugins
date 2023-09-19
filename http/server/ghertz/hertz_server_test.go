package ghertz

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	hertzServer "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/skirrund/gcloud/server"
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
			//t := &Test{}
			s := GetCookie("test", ctx)
			ctx.JSON(http.StatusOK, s)
		})
	})
	srv.Run(func() {
		fmt.Println("shut down")
	})
}
