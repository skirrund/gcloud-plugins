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
	fmt.Println(resp, err)
}
