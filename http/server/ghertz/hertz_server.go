package ghertz

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/network/standard"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/skirrund/gcloud-plugins/http/server/ghertz/middleware"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/response"
	gServer "github.com/skirrund/gcloud/server"
	"github.com/skirrund/gcloud/server/http/cookie"
	"github.com/skirrund/gcloud/utils/validator"
)

type Server struct {
	Svr     *server.Hertz
	Options gServer.Options
}

const (
	CookieDeleteMe     = "DeleteMe"
	CookieDeleteMaxAge = 0
	CookieDeleteVal    = ""
)

var CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

func NewServer(options gServer.Options, routerProvider func(engine *server.Hertz), middlewares ...app.HandlerFunc) gServer.Server {
	srv := &Server{}
	srv.Options = options
	var opts []config.Option
	opts = append(opts, server.WithMaxRequestBodySize(100*1024*1024))
	opts = append(opts, server.WithReadTimeout(5*time.Minute))
	opts = append(opts, server.WithWriteTimeout(5*time.Minute))
	opts = append(opts, server.WithHostPorts(options.Address))
	server.WithTransport(standard.NewTransporter)
	if options.IdleTimeout > 0 {
		opts = append(opts, server.WithIdleTimeout(options.IdleTimeout))
	}
	s := server.New(opts...)
	s.Name = options.ServerName
	s.Use(middleware.LoggingMiddleware, recovery.Recovery(recovery.WithRecoveryHandler(middleware.MyRecoveryHandler)))
	if len(middlewares) > 0 {
		s.Use(middlewares...)
	}
	routerProvider(s)
	srv.Svr = s
	return srv

}

func (server *Server) Shutdown() {
	server.Svr.Shutdown(context.Background())
}

func (server *Server) GetServeServer() any {
	return server.Svr
}

func (server *Server) Run(graceful ...func()) {
	// srv := &http.Server{
	// 	Addr:         server.Options.Address,
	// 	Handler:      server.Srv,
	// 	ReadTimeout:  60 * time.Second,
	// 	WriteTimeout: 60 * time.Second,
	// }
	server.Svr.Engine.OnShutdown = append(server.Svr.Engine.OnShutdown, func(ctx context.Context) {
		grace(server, graceful...)
		logger.Info("[Hertz]server has been shutdown")
	})
	server.Svr.SetCustomSignalWaiter(waitSignal)
	logger.Info("[Hertz] server starting on:", server.Options.Address)
	server.Svr.Spin()
}

func waitSignal(errCh chan error) error {
	signalToNotify := []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM}
	if signal.Ignored(syscall.SIGHUP) {
		signalToNotify = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, signalToNotify...)
	// graceful shutdown
	select {
	case sig := <-signals:
		logger.Info("Received signal:", sig)
		switch sig {
		case syscall.SIGTERM:
			return nil
		case syscall.SIGHUP, syscall.SIGINT:
			return nil
		}
	case err := <-errCh:
		// error occurs, exit immediately
		return err
	}
	return nil
}

func grace(server *Server, g ...func()) {
	server.Shutdown()
	for _, f := range g {
		f()
	}
}

// tag名称及使用方法
// path	绑定 url 上的路径参数，相当于 hertz 路由{:param}或{*param}中拿到的参数。例如：如果定义的路由为：/v:version/example，可以把 path 的参数指定为路由参数：path:"version"，此时，url: http://127.0.0.1:8888/v1/example，可以绑定path参数"1"
// form	绑定请求的 body 内容。content-type -> multipart/form-data 或 application/x-www-form-urlencoded，绑定 form 的 key-value
// query	绑定请求的 query 参数
// cookie	绑定请求的 cookie 参数
// header	绑定请求的 header 参数
// json	绑定请求的 body 内容 content-type -> application/json，绑定 json 参数
// raw_body	绑定请求的原始 body(bytes)，绑定的字段名不指定，也能绑定参数。（注：raw_body 绑定优先级最低，当指定多个 tag 时，一旦其他 tag 成功绑定参数，则不会绑定 body 内容。）
// 参数绑定优先级
// path > form > query > cookie > header > json > raw_body
func ShouldBind(ctx *app.RequestContext, obj any) error {
	err := ctx.Bind(obj)
	if err != nil {
		return err
	}
	err = validator.ValidateStruct(obj)
	if err != nil {
		return errors.New(validator.ErrResp(err))
	}
	return nil
}

func GetHeader(ctx *app.RequestContext, key string) string {
	return string(ctx.GetHeader(key))
}

func CheckQueryParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *app.RequestContext) bool {
	str := ctx.Query(name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckHeaderParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *app.RequestContext) bool {
	str := GetHeader(ctx, name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckParamsWithErrorMsg(name string, str string, v *string, errorMsg string, ctx *app.RequestContext) bool {
	*v = str
	if len(str) == 0 {
		if len(errorMsg) == 0 {
			ctx.JSON(http.StatusOK, response.ValidateError[any](name+"不能为空"))
		} else {
			ctx.JSON(http.StatusOK, response.ValidateError[any](errorMsg))
		}
		return false
	}
	return true
}

func CheckPostFormParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *app.RequestContext) bool {
	str, _ := ctx.GetPostForm(name)
	if len(str) == 0 {
		str = ctx.Query(name)
	}
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckQueryParams(name string, v *string, ctx *app.RequestContext) bool {
	return CheckQueryParamsWithErrorMsg(name, v, "", ctx)
}

func CheckPostFormParams(name string, v *string, ctx *app.RequestContext) bool {
	return CheckPostFormParamsWithErrorMsg(name, v, "", ctx)
}

func CheckHeaderParams(name string, v *string, ctx *app.RequestContext) bool {
	return CheckHeaderParamsWithErrorMsg(name, v, "", ctx)
}
func QueryArray(ctx *app.RequestContext, name string) []string {
	var params []string
	ctx.VisitAllQueryArgs(func(key, value []byte) {
		if string(key) == name {
			if len(value) > 0 {
				str := string(value)
				if strings.Contains(str, ",") {
					tmp := strings.Split(str, ",")
					params = append(params, tmp...)
				} else {
					params = append(params, str)
				}
			}
		}
	})
	return params
}
func PostFormArray(ctx *app.RequestContext, name string) []string {
	var params []string
	ctx.VisitAllPostArgs(func(key, value []byte) {
		if string(key) == name {
			if len(value) > 0 {
				str := string(value)
				if strings.Contains(str, ",") {
					tmp := strings.Split(str, ",")
					params = append(params, tmp...)
				} else {
					params = append(params, str)
				}
			}
		}
	})
	if len(params) > 0 {
		return params
	} else {
		return QueryArray(ctx, name)
	}
}

func GetCookie(name string, ctx *app.RequestContext) string {
	val := string(ctx.Cookie(name))
	if len(val) > 0 {
		val, _ = url.QueryUnescape(val)
	}
	return val
}

// del cookie
// if len(keys)==0 this function will delete all cookies
func ClearCookie(ctx *app.RequestContext, domain string, path string, keys ...string) {
	cookies := ctx.Request.Header.Cookies()
	l := len(keys)
	if len(cookies) > 0 {
		for _, c := range cookies {
			dm := string(c.Domain())
			if len(domain) > 0 {
				dm = domain
			}
			p := string(c.Path())
			if len(path) > 0 {
				p = path
			}
			name := string(c.Key())
			if len(name) > 0 {
				if l == 0 {
					cookie := protocol.AcquireCookie()
					defer protocol.ReleaseCookie(cookie)
					cookie.SetKey(name)
					cookie.SetValue(CookieDeleteVal)
					cookie.SetMaxAge(CookieDeleteMaxAge)
					cookie.SetPath(p)
					cookie.SetDomain(dm)
					cookie.SetSecure(c.Secure())
					cookie.SetHTTPOnly(c.HTTPOnly())
					cookie.SetSameSite(c.SameSite())
					cookie.SetExpire(time.Now().Add(-1 * time.Second))
					ctx.Response.Header.SetCookie(cookie)
					//ctx.SetCookie(name, CookieDeleteVal, CookieDeleteMaxAge, p, dm, c.SameSite(), c.Secure(), c.HTTPOnly())
				} else {
					for _, k := range keys {
						if name == k {
							cookie := protocol.AcquireCookie()
							defer protocol.ReleaseCookie(cookie)
							cookie.SetKey(name)
							cookie.SetValue(CookieDeleteVal)
							cookie.SetMaxAge(CookieDeleteMaxAge)
							cookie.SetPath(p)
							cookie.SetDomain(dm)
							cookie.SetSecure(c.Secure())
							cookie.SetHTTPOnly(c.HTTPOnly())
							cookie.SetSameSite(c.SameSite())
							cookie.SetExpire(time.Now().Add(-1 * time.Second))
							ctx.Response.Header.SetCookie(cookie)
							//ctx.SetCookie(name, CookieDeleteVal, -1, p, domain, c.SameSite(), c.Secure(), c.HTTPOnly())
						}
					}
				}
			}
		}
	}
}

func SetCookie(c cookie.Cookie, ctx *app.RequestContext) {
	if len(c.Key) > 0 {
		ctx.SetCookie(c.Key, c.Value, c.MaxAge, c.Path, c.Domain, getSameSite(c.SameSite), c.Secure, c.HttpOnly)
	}
}

func getSameSite(sameSite cookie.CookieSameSite) protocol.CookieSameSite {
	switch sameSite {
	case cookie.CookieSameSiteLaxMode:
		return protocol.CookieSameSiteLaxMode
	case cookie.CookieSameSiteStrictMode:
		return protocol.CookieSameSiteStrictMode
	case cookie.CookieSameSiteNoneMode:
		return protocol.CookieSameSiteNoneMode
	}
	return protocol.CookieSameSiteDefaultMode
}
