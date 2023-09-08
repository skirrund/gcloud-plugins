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

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/server/request"
	gResp "github.com/skirrund/gcloud/server/response"
)

type hertzHttpClient struct {
	client *client.Client
}

var defaultHttpClient hertzHttpClient

const (
	DefaultTimeout = 30 * time.Second
	ContentType    = "Content-Type"
	RequestTimeOut = 5 * time.Minute
	WriteTimeout   = RequestTimeOut
)

func NewClient() *hertzHttpClient {
	return &hertzHttpClient{}
}

func (hhc *hertzHttpClient) SetClient(c *client.Client) {
	hhc.client = c
}
func GetDefaultClient() hertzHttpClient {
	return defaultHttpClient
}

func init() {
	defaultHttpClient = hertzHttpClient{}
	defaultHttpClient.client, _ = client.NewClient(
		client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
		client.WithMaxConnDuration(DefaultTimeout),
		client.WithMaxIdleConnDuration(DefaultTimeout),
		client.WithMaxConnWaitTimeout(5*time.Second),
		client.WithClientReadTimeout(RequestTimeOut),
		client.WithWriteTimeout(WriteTimeout),
	)
}

func (hhc hertzHttpClient) getClient() *client.Client {
	if hhc.client == nil {
		return defaultHttpClient.client
	}
	return hhc.client
}

func (hhc hertzHttpClient) Exec(req *request.Request) (r *gResp.Response, err error) {
	doRequest := protocol.AcquireRequest()
	defer protocol.ReleaseRequest(doRequest)
	response := protocol.AcquireResponse()
	defer protocol.ReleaseResponse(response)
	reqUrl := req.Url
	r = &gResp.Response{
		Cookie:  make(map[string]string),
		Headers: make(map[string]string),
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
	if timeOut == 0 {
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
	cookie := protocol.AcquireCookie()
	defer protocol.ReleaseCookie(cookie)
	response.Header.VisitAllCookie(func(key, value []byte) {
		cookie.ParseBytes(value)
		val := string(cookie.Value())
		val, _ = url.QueryUnescape(val)
		r.Cookie[string(key)] = val
	})
	respHeaders := response.Header.GetHeaders()
	for _, h := range respHeaders {
		r.Headers[string(h.GetKey())] = string(h.GetValue())
	}
	if sc != http.StatusOK {
		logger.Error("[lb-heartz-client] StatusCode error:", sc, "【", reqUrl, "】", string(b), "【", string(response.Header.PeekLocation()), "】")
		return r, errors.New("heartz-client code error:" + strconv.FormatInt(int64(sc), 10))
	}
	return r, nil
}

func setHttpHeader(req *protocol.Request, headers map[string]string) {
	if headers == nil {
		return
	}
	for k, v := range headers {
		req.SetHeader(k, v)
	}
}

func (hertzHttpClient) CheckRetry(err error, status int) bool {
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
