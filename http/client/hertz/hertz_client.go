package hertz

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/server/http/cookie"
	"github.com/skirrund/gcloud/server/request"
	gResp "github.com/skirrund/gcloud/server/response"
)

type HertzHttpClient struct {
	client *client.Client
}

var defaultHttpClient HertzHttpClient

const (
	DefaultTimeout = 30 * time.Second
	ContentType    = "Content-Type"
	RequestTimeOut = 5 * time.Minute
	WriteTimeout   = RequestTimeOut
)

func (hhc *HertzHttpClient) SetClient(c *client.Client) {
	hhc.client = c
}
func GetDefaultClient() HertzHttpClient {
	return defaultHttpClient
}

func init() {
	defaultHttpClient = HertzHttpClient{}
	defaultHttpClient.client, _ = client.NewClient(
		client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
		client.WithMaxConnDuration(DefaultTimeout),
		client.WithMaxIdleConnDuration(DefaultTimeout),
		client.WithMaxConnWaitTimeout(5*time.Second),
		client.WithClientReadTimeout(RequestTimeOut),
		client.WithWriteTimeout(WriteTimeout),
	)
}

func (hhc HertzHttpClient) getClient() *client.Client {
	if hhc.client == nil {
		return defaultHttpClient.client
	}
	return hhc.client
}

func (hhc HertzHttpClient) Exec(req *request.Request) (r *gResp.Response, err error) {
	doRequest := protocol.AcquireRequest()
	defer protocol.ReleaseRequest(doRequest)
	response := protocol.AcquireResponse()
	defer protocol.ReleaseResponse(response)
	reqUrl := req.Url
	r = &gResp.Response{
		Cookies: make(map[string]*cookie.Cookie),
		Headers: make(map[string][]string),
	}
	if len(reqUrl) == 0 {
		return r, errors.New("[lb-heartz-client] request url  is empty")
	}
	params := req.Params
	headers := req.Headers
	isJson := req.IsJson
	doRequest.Header.SetMethod(req.Method)
	doRequest.SetRequestURI(reqUrl)
	defer func() {
		if err := recover(); err != nil {
			logger.Error("[lb-heartz-client] recover :", err)
		}
	}()
	if req.Method != http.MethodGet && req.Method != http.MethodHead && params != nil {
		bodyBytes, _ := io.ReadAll(params)
		doRequest.SetBody(bodyBytes)
		if isJson {
			doRequest.SetHeader(ContentType, "application/json;charset=utf-8")
		} else if req.HasFile {

		} else {
			doRequest.SetHeader(ContentType, "application/x-www-form-urlencoded;charset=utf-8")
		}
	}
	setHttpHeader(doRequest, headers)
	timeOut := req.TimeOut
	if timeOut <= 0 {
		timeOut = DefaultTimeout
	}
	doRequest.SetOptions(config.WithRequestTimeout(timeOut))
	err = hhc.getClient().Do(context.Background(), doRequest, response)
	if err != nil {
		logger.Error("[lb-heartz-client] DoRedirects error:", err.Error(), ",", reqUrl, ",")
		return r, err
	}
	sc := response.StatusCode()
	r.StatusCode = sc
	ct := string(response.Header.ContentType())
	r.ContentType = ct
	logger.Info("[lb-heartz-client] response statusCode:", sc, " content-type:", ct)
	b := response.Body()
	r.Body = b
	ck := protocol.AcquireCookie()
	defer protocol.ReleaseCookie(ck)
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
		vals = append(vals, val)
		r.Headers[k] = vals
	})
	if sc != http.StatusOK {
		logger.Error("[lb-heartz-client] StatusCode error:", sc, "【", reqUrl, "】", string(b), "【", string(response.Header.PeekLocation()), "】")
		return r, errors.New("heartz-client code error:" + strconv.FormatInt(int64(sc), 10))
	}
	return r, nil
}

func (hhc HertzHttpClient) Proxy(targetUrl string, ctx *app.RequestContext, timeout time.Duration) error {
	logger.Info("[startProxy-hertz]:", targetUrl)
	creq := protocol.AcquireRequest()
	defer protocol.ReleaseRequest(creq)
	var err error
	method := string(ctx.Method())
	creq.Header.SetMethod(method)
	if method != http.MethodGet && method != http.MethodHead {
		creq.SetBody(ctx.Request.Body())
	}
	ctx.VisitAllHeaders(func(key, value []byte) {
		creq.Header.SetBytesKV(key, value)
	})
	creq.Header.SetContentTypeBytes(ctx.ContentType())
	creq.SetRequestURI(targetUrl)
	resp := protocol.AcquireResponse()
	defer protocol.ReleaseResponse(resp)
	if timeout <= 0 {
		timeout = RequestTimeOut
	}
	creq.SetOptions(config.WithRequestTimeout(timeout))
	err = hhc.getClient().Do(context.Background(), creq, resp)
	if err != nil {
		return err
	}
	resp.Header.VisitAll(func(key, value []byte) {
		ctx.Response.Header.SetBytesV(string(key), value)
	})
	sc := resp.StatusCode()
	ctx.SetStatusCode(sc)
	ctx.Response.SetBody(resp.Body())
	return nil
}

func getSameSite(sameSite protocol.CookieSameSite) (s cookie.CookieSameSite) {
	switch sameSite {
	case protocol.CookieSameSiteDefaultMode:
		return
	case protocol.CookieSameSiteLaxMode:
		s = cookie.CookieSameSiteLaxMode
	case protocol.CookieSameSiteStrictMode:
		s = cookie.CookieSameSiteStrictMode
	case protocol.CookieSameSiteNoneMode:
		s = cookie.CookieSameSiteNoneMode
	}
	return s
}

func setHttpHeader(req *protocol.Request, headers map[string]string) {
	if headers == nil {
		return
	}
	for k, v := range headers {
		req.SetHeader(k, v)
	}
}

func (HertzHttpClient) CheckRetry(err error, status int) bool {
	if err != nil {
		// if err == fasthttp.ErrDialTimeout {
		// 	return true
		// }
		ue, ok := err.(*url.Error)
		logger.Info("[lb-heartz-client] checkRetry error *url.Error:", ok)
		if ok {
			if ue.Err != nil {
				no, ok := ue.Err.(*net.OpError)
				if ok && no.Op == "dial" {
					return true
				}
			}
		} else {
			no, ok := err.(*net.OpError)
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
