package hertz

import (
	"fmt"
	"testing"

	"github.com/skirrund/gcloud/server/http"
	"github.com/skirrund/gcloud/server/lb"
)

func TestXxx(t *testing.T) {
	var r []byte
	lb.GetInstance().SetHttpClient(defaultHttpClient)
	resp, err := http.GetUrl("https://www.baidu.com", nil, nil, &r)
	fmt.Println(string(resp.Body), string(r))
	r = r[:0]
	fmt.Println(string(resp.Body), err)
}
