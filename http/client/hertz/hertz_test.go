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
	resp, err := http.PostJSONUrl("http://127.0.0.1:8080/test", nil, nil, &r)
	fmt.Println(resp, err)
}
