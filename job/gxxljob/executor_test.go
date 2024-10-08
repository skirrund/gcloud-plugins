package gxxljob

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/skirrund/gcloud/logger"
)

func TestXxx(t *testing.T) {
	opts := Options{
		AdminAddresses:   "http://xxljob:8089/xxl-job-admin",
		AppName:          "xxl-job-test-go",
		Logretentiondays: 1,
	}
	executor, err := RunWithOptions(opts)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	executor.RegTask("go-test", JobRun)
	var s string
	fmt.Scanln(&s)
	fmt.Println(s)
}
func JobRun(ctx context.Context, req *RunRequest) error {
	logger.Info("run job")
	return errors.New("test")
}
