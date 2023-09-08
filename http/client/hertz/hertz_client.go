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

type HertzHttpClient struct {
	hertzClient *client.Client
}

var hertzHttpClient HertzHttpClient

const (
	DefaultTimeout = 30 * time.Second
	ContentType    = "Content-Type"
	RequestTimeOut = 5 * time.Minute
	WriteTimeout   = RequestTimeOut
)

func (hhc *HertzHttpClient) SetClient(c *client.Client) {
	hhc.hertzClient = c
}

func init() {
	hertzHttpClient = HertzHttpClient{}
	hertzHttpClient.hertzClient, _ = client.NewClient(
		client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
		client.WithMaxConnDuration(DefaultTimeout),
		client.WithMaxIdleConnDuration(DefaultTimeout),
		client.WithMaxConnWaitTimeout(5*time.Second),
		client.WithClientReadTimeout(RequestTimeOut),
		client.WithWriteTimeout(WriteTimeout),
	)
}

func (hhc HertzHttpClient) getClient() *client.Client {
	if hhc.hertzClient == nil {
		return hertzHttpClient.hertzClient
	}
	return hhc.hertzClient
}

func (hhc HertzHttpClient) Exec(req *request.Request) (r *gResp.Response, err error) {
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

	err = hhc.getClient().DoRedirects(context.Background(), doRequest, response, 1)
	if err != nil {
		logger.Error("[lb-heartz-client] fasthttp.Do error:", err.Error(), ",", reqUrl, ",")
		return r, err
	}
	sc := response.StatusCode()
	r.StatusCode = sc
	ct := string(response.Header.ContentType())
	r.ContentType = ct
	logger.Info("[lb-heartz-client] response statusCode:", sc, " content-type:", ct)
	b := response.Body()
	r.Body = b
	if sc != http.StatusOK {
		logger.Error("[lb-heartz-client] StatusCode error:", sc, ",", reqUrl, ",", string(b))
		return r, errors.New("heartz-client code error:" + strconv.FormatInt(int64(sc), 10))
	}
	cookies := response.Header.GetCookies()
	for _, c := range cookies {
		r.Cookie[string(c.GetKey())] = string(c.GetValue())
	}
	respHeaders := response.Header.GetHeaders()
	for _, h := range respHeaders {
		r.Headers[string(h.GetKey())] = string(h.GetValue())
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
