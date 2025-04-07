package fasthttp

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/server/http/cookie"
	"github.com/skirrund/gcloud/server/lb"
	"github.com/skirrund/gcloud/server/request"
	gResp "github.com/skirrund/gcloud/server/response"
	"github.com/valyala/fasthttp"
)

type FastHttpClient struct {
	client *fasthttp.Client
}

var defaultClient FastHttpClient

const (
	DefaultTimeout  = 10 * time.Second
	default_timeout = 10 * time.Second
	RequestTimeOut  = 5 * time.Minute
)

func GetDefaultClient() FastHttpClient {
	return defaultClient
}

func (c *FastHttpClient) SetClient(client *fasthttp.Client) {
	c.client = client
}

func init() {
	defaultClient = FastHttpClient{}
	defaultClient.client = &fasthttp.Client{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, 3*time.Second)
		},
		MaxConnsPerHost:     2000,
		MaxIdleConnDuration: DefaultTimeout,
		MaxConnDuration:     DefaultTimeout,
		ReadTimeout:         5 * time.Minute,
		WriteTimeout:        5 * time.Minute,
		MaxConnWaitTimeout:  5 * time.Second,
	}
}

func (hhc FastHttpClient) getClient() *fasthttp.Client {
	if hhc.client == nil {
		return defaultClient.client
	}
	return hhc.client
}

func (c FastHttpClient) Exec(req *request.Request) (r *gResp.Response, err error) {
	doRequest := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(doRequest)
	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)
	reqUrl := req.Url
	r = &gResp.Response{
		Cookies: make(map[string]*cookie.Cookie),
		Headers: make(map[string][]string),
	}
	if len(reqUrl) == 0 {
		return r, errors.New("[lb-fasthttp] request url  is empty")
	}
	params := req.Params
	headers := req.Headers
	isJson := req.IsJson
	doRequest.Header.SetMethod(req.Method)
	doRequest.SetRequestURI(reqUrl)
	defer func() {
		if err := recover(); err != nil {
			logger.Error("[lb-fasthttp]] recover :", err)
		}
	}()
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		// bodyBytes, _ := io.ReadAll(params)
		doRequest.SetBody(params)
		if isJson {
			doRequest.Header.SetContentType("application/json;charset=utf-8")
		} else if req.HasFile {

		} else {
			doRequest.Header.SetContentType("application/x-www-form-urlencoded;charset=utf-8")
		}
	}
	setFasthttpHeader(doRequest, headers)
	timeOut := req.TimeOut
	if timeOut == 0 {
		timeOut = default_timeout
	}
	doRequest.SetTimeout(timeOut)
	err = c.getClient().Do(doRequest, response)
	if err != nil {
		logger.Error("[lb-fasthttp] fasthttp.Do error:", err.Error(), ",", reqUrl, ",")
		return r, err
	}
	sc := response.StatusCode()
	r.StatusCode = sc
	ct := string(response.Header.ContentType())
	r.ContentType = ct
	logger.Info("[lb-fasthttp] response statusCode:", sc, " content-type:", ct)
	var location string
	if sc >= http.StatusMultipleChoices && sc <= http.StatusPermanentRedirect {
		location = string(response.Header.Peek("Location"))
		// logger.Warn("[lb-fasthttp] DoRedirects{ statusCode:", sc, ",location:", location, "}")
		// if len(location) > 0 {
		// 	response.Reset()
		// 	doRequest.SetRequestURI(location)
		// 	err = fastClient.DoTimeout(doRequest, response, timeOut)
		// 	if err != nil {
		// 		logger.Error("[lb-fasthttp] DoRedirects error:", err.Error(), ",", reqUrl, ",")
		// 		return 0, err
		// 	}
		// 	sc = response.StatusCode()
		// 	ct = string(response.Header.ContentType())
		// 	logger.Info("[lb-fasthttp] DoRedirects response statusCode:", sc, " content-type:", ct)
		// }
	}
	b := response.Body()
	r.Body = b
	ck := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(ck)
	response.Header.VisitAllCookie(func(key, value []byte) {
		ck.ParseBytes(value)
		val := string(ck.Value())
		val, _ = url.QueryUnescape(val)
		k := string(key)
		r.Cookies[k] = &cookie.Cookie{
			Key:      k,
			Value:    val,
			Expire:   ck.Expire(),
			MaxAge:   ck.MaxAge(),
			Domain:   string(ck.Domain()),
			Path:     string(ck.Path()),
			HttpOnly: ck.HTTPOnly(),
			Secure:   ck.Secure(),
			SameSite: getSameSite(ck.SameSite()),
		}
	})
	response.Header.VisitAll(func(key, value []byte) {
		k := string(key)
		val := string(value)
		vals := r.Headers[k]
		logger.Info("[lb-fasthttp] header:", k, "=", val)
		vals = append(vals, val)
		r.Headers[k] = vals
	})
	if sc != http.StatusOK {
		logger.Error("[lb-fasthttp] StatusCode error:", sc, ",", reqUrl, ",", string(b), ",", location)
		return r, errors.New("fasthttp code error:" + strconv.FormatInt(int64(sc), 10))
	}
	return r, nil
}

func (c FastHttpClient) Proxy(targetUrl string, ctx *fiber.Ctx, timeout time.Duration) error {
	logger.Info("[startProxy-fasthttp]:", targetUrl)
	creq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(creq)
	var err error
	req := ctx.Request()
	ct := string(req.Header.ContentType())
	method := string(req.Header.Method())
	creq.Header.SetMethod(method)
	if method != http.MethodGet && method != http.MethodHead {
		creq.SetBody(req.Body())
	}
	for k, v := range ctx.GetReqHeaders() {
		length := len(v)
		for i := 0; i < length; i++ {
			if i == 0 {
				creq.Header.Set(k, v[i])
			} else {
				creq.Header.Add(k, v[i])
			}
		}
	}
	creq.Header.SetContentType(ct)
	creq.SetRequestURI(targetUrl)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	if timeout <= 0 {
		timeout = RequestTimeOut
	}
	creq.SetTimeout(timeout)
	err = c.getClient().Do(creq, resp)
	if err != nil {
		return err
	}
	resp.Header.VisitAll(func(key, value []byte) {
		ctx.Response().Header.SetBytesKV(key, value)
	})
	sc := resp.StatusCode()
	ctx.Status(sc).Response().SetBody(resp.Body())
	return nil
}

func (fhc FastHttpClient) ProxyService(serviceName, path string, ctx *fiber.Ctx, timeout time.Duration) error {
	logger.Info("[startProxy-fasthttp]:", serviceName, "[path:", path)
	gRequest := &request.Request{
		ServiceName: serviceName,
		Path:        path,
		Method:      ctx.Method(),
		TimeOut:     timeout,
	}
	// r := bytes.NewReader(ctx.Body())
	gRequest.Params = ctx.Body()
	ctxHeader := make(map[string]string)
	for k, v := range ctx.GetReqHeaders() {
		length := len(v)
		for i := range length {
			ctxHeader[k] = v[i]
		}
	}
	gRequest.Headers = ctxHeader
	gRequest.LbOptions = request.NewDefaultLbOptions()
	gresp, err := lb.GetInstance().Run(gRequest, nil)
	if err != nil {
		return err
	}
	fResp := ctx.Response()
	if len(gresp.Headers) > 0 {
		for k, v := range gresp.Headers {
			if len(v) == 0 {
				fResp.Header.Set(k, v[0])
			} else {
				for _, vv := range v {
					fResp.Header.Add(k, vv)
				}
			}
		}
	}
	sc := gresp.StatusCode
	ctx.Context().SetContentType(gresp.ContentType)
	ctx.Context().SetBody(gresp.Body)
	ctx.Context().SetStatusCode(sc)
	return nil
}

func getSameSite(sameSite fasthttp.CookieSameSite) (s cookie.CookieSameSite) {
	switch sameSite {
	case fasthttp.CookieSameSiteDefaultMode:
		return
	case fasthttp.CookieSameSiteLaxMode:
		s = cookie.CookieSameSiteLaxMode
	case fasthttp.CookieSameSiteStrictMode:
		s = cookie.CookieSameSiteStrictMode
	case fasthttp.CookieSameSiteNoneMode:
		s = cookie.CookieSameSiteNoneMode
	}
	return s
}

func setFasthttpHeader(req *fasthttp.Request, headers map[string]string) {
	if headers == nil {
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func (FastHttpClient) CheckRetry(err error, status int) bool {
	if err != nil {
		if errors.Is(err, fasthttp.ErrDialTimeout) {
			return true
		}
		err = errors.Unwrap(err)
		if err == nil {
			return false
		}
		ue, ok := err.(*url.Error)
		logger.Info("[lb-fasthttp] checkRetry error *url.Error:", ok)
		if ok {
			if ue.Err != nil {
				no, ok := ue.Err.(*net.OpError)
				logger.Info("[lb-fasthttp] checkRetry error url->*net.OpError:", ok)
				if ok && no.Op == "dial" {
					return true
				}
			}
		} else {
			no, ok := err.(*net.OpError)
			logger.Info("[lb-fasthttp] checkRetry error *net.OpError:", ok)
			if ok && no.Op == "dial" {
				return true
			}
		}
		if status == 404 || status == 502 || status == 504 {
			return true
		}
	}
	return false
}
