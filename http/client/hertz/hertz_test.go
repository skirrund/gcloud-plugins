package hertz

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/skirrund/gcloud/server/request"
	"golang.org/x/net/http2"
)

func TestXxx(t *testing.T) {
	var wg sync.WaitGroup
	start1 := time.Now()
	for i := 0; i < 1; i++ {
		strIdx := strconv.Itoa(i)
		wg.Go(func() {
			start := time.Now()
			resp, _ := defaultHttpClient.Exec(&request.Request{
				H2C:    false,
				Url:    "http://127.0.0.1:8899/test?" + strIdx,
				Method: "GET",
			})
			fmt.Println(string(resp.Body), "finished:", time.Since(start).Milliseconds(), "ms")
		})
	}
	wg.Wait()
	fmt.Println("total finished:", time.Since(start1).Milliseconds(), "ms")

}

func TestH2CGo(t *testing.T) {
	h2cTransport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
	// client := http.Client{
	// 	Transport: h2cTransport,
	// }
	// client = http.Client{
	// 	Transport: &http.Transport{},
	// }
	// var wg sync.WaitGroup
	start1 := time.Now()
	for i := 0; i < 1; i++ {
		// wg.Go(func() {
		start := time.Now()
		req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/test", nil)
		resp, err := h2cTransport.RoundTrip(req)
		b, _ := io.ReadAll(resp.Body)
		fmt.Println(string(b), resp.Proto, err)
		fmt.Println("finished:", time.Since(start).Milliseconds(), "ms")
		// })
	}
	// wg.Wait()

	fmt.Println("total finished:", time.Since(start1).Milliseconds(), "ms")
}
