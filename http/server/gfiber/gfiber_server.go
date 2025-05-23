package gfiber

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/skirrund/gcloud-plugins/http/server/gfiber/middleware"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/response"
	"github.com/skirrund/gcloud/server"
	"github.com/skirrund/gcloud/server/http/cookie"
	"github.com/skirrund/gcloud/utils"
	"github.com/skirrund/gcloud/utils/validator"
	"github.com/valyala/fasthttp"
)

type Server struct {
	Srv     *fiber.App
	Options server.Options
}

const (
	CookieDeleteMe     = "DeleteMe"
	CookieDeleteMaxAge = 0
)

var CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

func NewServer(options server.Options, routerProvider func(engine *fiber.App), middleware ...any) server.Server {
	srv := &Server{}
	srv.Options = options
	bodySize := options.MaxRequestBodySize
	if bodySize <= 0 {
		bodySize = 100 * 1024 * 1024
	}
	cfg := fiber.Config{
		Concurrency:  options.Concurrency,
		BodyLimit:    bodySize,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if e, ok := err.(*fiber.Error); ok {
				if e.Code == fiber.StatusInternalServerError {
					c.JSON(response.Fail[any](err.Error()))
				} else {
					c.SendStatus(e.Code)
				}
			} else if e, ok := err.(*server.Error); ok {
				resp := response.Response[any]{
					Code:       e.Code,
					Message:    e.Msg,
					SubMessage: e.SubMsg,
					Success:    false,
				}
				c.JSON(resp)
			} else if e, ok := err.(server.Error); ok {
				resp := response.Response[any]{
					Code:       e.Code,
					Message:    e.Msg,
					SubMessage: e.SubMsg,
					Success:    false,
				}
				c.JSON(resp)
			} else {
				logger.Error("[Fiber] error:", err, "\n", string(debug.Stack()))
			}
			return nil
		},
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	}
	s := fiber.New(cfg)
	hanlders := getCfg()
	s.Use(hanlders...)
	if len(middleware) > 0 {
		s.Use(middleware...)
	}
	routerProvider(s)
	s.Use("/", func(c *fiber.Ctx) error {
		c.SendStatus(fiber.StatusNotFound)
		return nil
	})
	srv.Srv = s
	return srv
}
func getCfg() []any {
	var handlers []any
	recoverCfg := recover.Config{
		Next: func(c *fiber.Ctx) bool {
			return false
		},
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e any) {
			logger.Error("[Fiber] recover:", e, "\n", string(debug.Stack()))
			str, _ := utils.MarshalToString(e)
			c.JSON(response.Fail[any](str))
		},
	}
	handlers = append(handlers, recover.New(recoverCfg), middleware.LoggingMiddleware)
	return handlers
}

func (server *Server) Shutdown() {
}

func (server *Server) GetServeServer() any {
	return server.Srv
}

func (server *Server) Run(graceful ...func()) {
	// srv := &http.Server{
	// 	Addr:         server.Options.Address,
	// 	Handler:      server.Srv,
	// 	ReadTimeout:  60 * time.Second,
	// 	WriteTimeout: 60 * time.Second,
	// }
	go func() {
		logger.Info("[Fiber] server starting on:", server.Options.Address)
		if err := server.Srv.Listen(server.Options.Address); err != nil && err != http.ErrServerClosed {
			logger.Panic("[Fiber] listen:", err.Error())
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("[Fiber]Shutting down server...")
	if err := server.Srv.ShutdownWithTimeout(10 * time.Second); err != nil {
		grace(server, graceful...)
		logger.Panic("[Fiber]Server forced to shutdown:", err)
	} else {
		grace(server, graceful...)
	}
	logger.Info("[Fiber]server has been shutdown")
}

func grace(server *Server, g ...func()) {
	server.Shutdown()
	for _, f := range g {
		f()
	}
}

// ShouldBindBody binds the request body to a struct.
// It supports decoding the following content types based on the Content-Type header:
// application/json, application/xml, application/x-www-form-urlencoded, multipart/form-data
// If none of the content types above are matched, it will return a ErrUnprocessableEntity error
func ShouldBindBody(ctx *fiber.Ctx, obj any) error {
	err := ctx.BodyParser(obj)
	if err != nil {
		return err
	}
	err = validator.ValidateStruct(obj)
	if err != nil {
		return errors.New(validator.ErrResp(err))
	}
	return nil
}

func ShouldBindParams(ctx *fiber.Ctx, obj any) error {
	err := ctx.ParamsParser(obj)
	if err != nil {
		return err
	}
	err = validator.ValidateStruct(obj)
	if err != nil {
		return errors.New(validator.ErrResp(err))
	}
	return nil
}

func ShouldBindQuery(ctx *fiber.Ctx, obj any) error {
	err := ctx.QueryParser(obj)
	if err != nil {
		return err
	}
	err = validator.ValidateStruct(obj)
	if err != nil {
		return errors.New(validator.ErrResp(err))
	}
	return nil
}
func ShouldBindHeader(ctx *fiber.Ctx, obj any) error {
	err := ctx.ReqHeaderParser(obj)
	if err != nil {
		return err
	}
	err = validator.ValidateStruct(obj)
	if err != nil {
		return errors.New(validator.ErrResp(err))
	}
	return nil
}

func GetHeader(ctx *fiber.Ctx, key string) string {
	return string(ctx.Request().Header.Peek(key))
}

func CheckQueryParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *fiber.Ctx) bool {
	str := ctx.Query(name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckHeaderParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *fiber.Ctx) bool {
	str := GetHeader(ctx, name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckParamsWithErrorMsg(name string, str string, v *string, errorMsg string, ctx *fiber.Ctx) bool {
	*v = str
	if len(str) == 0 {
		if len(errorMsg) == 0 {
			ctx.JSON(response.ValidateError[any](name + "不能为空"))
		} else {
			ctx.JSON(response.ValidateError[any](errorMsg))
		}
		return false
	}
	return true
}

func CheckPostFormParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *fiber.Ctx) bool {
	str := ctx.FormValue(name)
	if len(str) == 0 {
		str = ctx.Query(name)
	}
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckQueryParams(name string, v *string, ctx *fiber.Ctx) bool {
	return CheckQueryParamsWithErrorMsg(name, v, "", ctx)
}

func CheckPostFormParams(name string, v *string, ctx *fiber.Ctx) bool {
	return CheckPostFormParamsWithErrorMsg(name, v, "", ctx)
}

func CheckHeaderParams(name string, v *string, ctx *fiber.Ctx) bool {
	return CheckHeaderParamsWithErrorMsg(name, v, "", ctx)
}
func QueryArray(ctx *fiber.Ctx, name string) []string {
	array := ctx.Context().QueryArgs().PeekMulti(name)
	var params []string
	if len(array) > 0 {
		for _, a := range array {
			if len(a) == 0 {
				continue
			}
			v := string(a)
			if strings.Contains(v, ",") {
				tmp := strings.Split(v, ",")
				params = append(params, tmp...)
			} else {
				params = append(params, v)
			}
		}
	}
	return params
}
func PostFormArray(ctx *fiber.Ctx, name string) []string {
	array := ctx.Context().PostArgs().PeekMulti(name)
	var params []string
	if len(array) > 0 {
		for _, a := range array {
			if len(a) == 0 {
				continue
			}
			v := string(a)
			if strings.Contains(v, ",") {
				tmp := strings.Split(v, ",")
				params = append(params, tmp...)
			} else {
				params = append(params, v)
			}
		}
	}
	if len(params) > 0 {
		return params
	} else {
		return QueryArray(ctx, name)
	}
}

func GetCookie(name string, ctx *fiber.Ctx) string {
	val := ctx.Cookies(name)
	if len(val) > 0 {
		val, _ = url.QueryUnescape(val)
	}
	return val
}

// del cookie
// if len(keys)==0 this function will delete all cookies
func ClearCookie(ctx *fiber.Ctx, domain string, path string, keys ...string) {
	length := len(keys)
	ctx.Request().Header.VisitAllCookie(func(key, value []byte) {
		k := string(key)
		if len(key) > 0 {
			c := fasthttp.AcquireCookie()
			defer fasthttp.ReleaseCookie(c)
			err := c.ParseBytes(value)
			fCookie := &fiber.Cookie{
				Name:     k,
				Value:    CookieDeleteMe,
				Path:     string(c.Path()),
				Domain:   string(c.Domain()),
				MaxAge:   CookieDeleteMaxAge,
				Expires:  CookieExpireDelete,
				Secure:   c.Secure(),
				HTTPOnly: c.HTTPOnly(),
			}
			if len(domain) > 0 {
				fCookie.Domain = domain
			}
			if len(path) > 0 {
				fCookie.Path = path
			}
			if err == nil {
				if length == 0 {
					ctx.Cookie(fCookie)
				} else {
					for _, nk := range keys {
						if nk == k {
							ctx.Cookie(fCookie)
						}
					}
				}
			}
		}
	})
}

func SetCookie(c cookie.Cookie, ctx *fiber.Ctx) {
	if len(c.Key) <= 0 {
		return
	}
	fCookie := &fiber.Cookie{
		Name:     c.Key,
		Value:    url.QueryEscape(c.Value),
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   c.MaxAge,
		Secure:   c.Secure,
		HTTPOnly: c.HttpOnly,
	}
	if len(c.SameSite) > 0 {
		fCookie.SameSite = string(c.SameSite)
	}
	ctx.Cookie(fCookie)
}
