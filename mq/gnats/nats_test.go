package gnats

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/skirrund/gcloud/mq"
)

func TestDemo1(t *testing.T) {
	user := "hmb"
	//user = "pbm"
	pwd := "admin"
	// nc, err := nats.Connect("nats://nats2:4222, nats://nats1:4223,nats://nats3:4224", nats.Name("test"), nats.UserInfo(user, pwd))

	conn, err := NewConnection(Option{
		Url:      "nats://nats2:4222, nats://nats1:4223,nats://nats3:4224",
		AppName:  "test",
		User:     user,
		Password: pwd,
	})
	if err != nil {
		panic(err)
	}
	//subject := "public/common-base/wx"
	// subject = "test-3-push"

	subject := "test-1-2"
	//var wg sync.WaitGroup
	for i := 0; i != 10; i++ {
		// subject := "test-schedule.schedules." + utils.Uuid()
		header := map[string]string{}
		// header["Nats-Schedule-Target"] = "test-schedule-test-1"
		// header["Nats-Schedule"] = "@at " + time.Now().Add(3*time.Second).Format(time.RFC3339)
		// uuid := utils.Uuid()
		// header["Nats-Msg-Id"] = uuid
		// fmt.Println(uuid)
		data := map[string]any{"k1": strconv.Itoa(i), "v1": strconv.Itoa(i), "time": time.Now().Format(time.DateTime)}
		str, _ := sonic.MarshalString(data)
		msg := &mq.Message{
			Topic:   subject,
			Payload: []byte(str),
			Header:  header,
			//	NatsOpts: mq.NatsOpts{Stream: "test-schedule"},
			//DeliverAfter: 10 * time.Second,
		}
		// wg.Go(func() {
		err = conn.Send(msg)
		fmt.Println(err)
		// })
	}
	//	wg.Wait()

}

func TestConsumer(t *testing.T) {
	user := "hmb"
	//user = "pbm"
	pwd := "admin"
	// nc, err := nats.Connect("nats://nats2:4222, nats://nats1:4223,nats://nats3:4224", nats.Name("test"), nats.UserInfo(user, pwd))

	conn, err := NewConnection(Option{
		Url:      "nats://nats2:4222, nats://nats1:4223,nats://nats3:4224",
		AppName:  "test",
		User:     user,
		Password: pwd,
	})
	if err != nil {
		panic(err)
	}
	subject := "public-common-base-wx"
	subject = "test-1-2"
	// conn.Subscribe(mq.ConsumerOptions{
	// 	Topic:            subject,
	// 	SubscriptionName: "test-schedule-test-1",
	// 	MessageListener:  OnMessage,
	// 	NatsOpts:         mq.NatsOpts{Stream: "test-schedule", PullBatchSize: 1},
	// 	IsErrorPanic:     true,
	// 	ACKMode:          mq.ACK_MANUAL,
	// })
	conn.SubscribeSync(mq.ConsumerOptions{
		Topic:            subject,
		SubscriptionName: "test-1",
		MessageListener:  OnMessage1,
		//NatsOpts:         mq.NatsOpts{Stream: "test-schedule"},
		IsErrorPanic: true,
		ACKMode:      mq.ACK_MANUAL,
	})
}

func OnMessage(ctx context.Context, msg *mq.Message) error {
	fmt.Println("OnMessage:", msg.SubOpts.Name, ","+string(msg.Payload))
	return nil
}

func OnMessage1(ctx context.Context, msg *mq.Message) error {
	fmt.Println("OnMessage1:", msg.SubOpts.Name, ","+string(msg.Payload), time.Now().String())
	return nil
}
func TestTTT(test *testing.T) {
	fmt.Println(time.Now().Format(time.RFC3339))
}
