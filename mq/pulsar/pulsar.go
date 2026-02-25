package pulsar

import (
	"errors"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/mq"
	"github.com/skirrund/gcloud/tracer"
	"github.com/skirrund/gcloud/utils"

	"github.com/apache/pulsar-client-go/pulsar"
)

type Message struct {
	Msg pulsar.Message
}

type PulsarClient struct {
	client  pulsar.Client
	appName string
	mt      sync.Mutex
}

const (
	SERVER_URL_KEY           = "pulsar.service-url"
	CONNECTION_TIMEOUT_KEY   = "pulsar.connectionTimeout"
	OPERATION_TIMEOUT_KEY    = "pulsar.operationTimeout"
	defaultConnectionTimeout = 5 * time.Second
	defaultOperationTimeout  = 30 * time.Second
)

const (
	MAX_RETRY_TIMES = 50
)

var pulsarClient *PulsarClient
var once sync.Once

func init() {
	os.Setenv("GODEBUG", "urlstrictcolons=0")
}

func NewClient(url string, connectionTimeoutSecond int64, operationTimeoutSecond int64, appName string) mq.IClient {
	if pulsarClient != nil {
		return pulsarClient
	}
	once.Do(func() {
		pulsarClient = &PulsarClient{
			client:  createClient(url, connectionTimeoutSecond, operationTimeoutSecond),
			appName: appName,
		}
	})
	return pulsarClient

}

func createClient(url string, connectionTimeoutSecond int64, operationTimeoutSecond int64) pulsar.Client {
	var cts time.Duration
	var ots time.Duration
	if connectionTimeoutSecond > 0 {
		cts = time.Duration(connectionTimeoutSecond) * time.Second
	} else {
		cts = defaultConnectionTimeout
	}
	if operationTimeoutSecond > 0 {
		ots = time.Duration(operationTimeoutSecond) * time.Second
	} else {
		ots = defaultOperationTimeout
	}
	logger.Infof("[pulsar]start init pulsar-client:" + url)
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:                     url,
		ConnectionTimeout:       cts,
		OperationTimeout:        ots,
		MaxConnectionsPerBroker: runtime.NumCPU(),
	})
	if err != nil {
		panic(err)
	}
	logger.Infof("[pulsar]finished init pulsar-client")
	return client
}

func getAppName(appName string) string {
	name := utils.Uuid()
	if len(appName) == 0 {
		return name
	} else {
		return appName + "-" + name
	}
}

func (pc *PulsarClient) Send(msg *mq.Message) error {
	return pc.doSend(msg)
}

func (pc *PulsarClient) SendAsync(msg *mq.Message) error {
	return pc.doSendAsync(msg)
}

func (pc *PulsarClient) Subscribe(opts mq.ConsumerOptions) error {
	go pc.doSubscribe(opts)
	return nil
}
func (pc *PulsarClient) SubscribeSync(opts mq.ConsumerOptions) error {
	return pc.doSubscribe(opts)
}

func (pc *PulsarClient) doSubscribe(opts mq.ConsumerOptions) error {
	subscriptionName := opts.SubscriptionName
	topic := opts.Topic
	logger.Infof("[pulsar]ConsumerOptions:%+v", opts)
	options := pulsar.ConsumerOptions{
		Topic:               topic,
		SubscriptionName:    subscriptionName,
		Type:                pulsar.SubscriptionType(opts.SubscriptionType),
		Name:                getAppName(pc.appName),
		NackRedeliveryDelay: 15 * time.Second,
		//Schema:              pulsar.NewStringSchema(nil),
	}
	if opts.RetryTimes == 0 {
		opts.RetryTimes = MAX_RETRY_TIMES
	}
	schema := pulsar.NewJSONSchema(`"string"`, nil)
	channelSize := opts.MaxMessageChannelSize
	if channelSize == 0 {
		channelSize = 200
	}
	channel := make(chan pulsar.ConsumerMessage, channelSize)
	options.MessageChannel = channel
	consumer, err := pc.client.Subscribe(options)
	if err != nil {
		logger.Error(errors.New("[pulsar] Subscribe error:" + err.Error()))
		if opts.IsErrorPanic {
			panic("[pulsar] Subscribe error:" + err.Error())
		} else {
			return err
		}
	}
	logger.Infof("[pulsar]store consumerOptions:"+topic+":"+subscriptionName+",%+v", opts)

	for cm := range channel {
		go func(cm pulsar.ConsumerMessage, consumer pulsar.Consumer, schema pulsar.Schema, opts mq.ConsumerOptions) {
			consume(cm, consumer, schema, opts)
		}(cm, consumer, schema, opts)
	}
	return nil
}
func consume(cm pulsar.ConsumerMessage, consumer pulsar.Consumer, schema pulsar.Schema, opts mq.ConsumerOptions) {
	logCtx := tracer.NewTraceIDContext()
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorContext(logCtx, "[pulsar] consumer panic recover :", err, "\n", string(debug.Stack()))
		}
	}()
	msg := cm.Message
	var msgStr string
	logger.InfofContext(logCtx, "[pulsar] consumer info=>subName:%s,msgId:%v,reDeliveryCount:%d,publishTime:%v,producerName:%s", cm.Subscription(), msg.ID(), msg.RedeliveryCount(), msg.PublishTime(), msg.ProducerName())
	err := schema.Decode(msg.Payload(), &msgStr)
	if err != nil {
		logger.InfoContext(logCtx, "[pulsar] consumer msg:", err.Error())
	} else {
		logger.InfoContext(logCtx, "[pulsar] consumer msg:", msgStr)
	}
	err = opts.MessageListener(logCtx, &mq.Message{
		Payload:         []byte(msgStr),
		RedeliveryCount: uint64(msg.RedeliveryCount()),
		SubOpts:         mq.SubOpts{Name: opts.SubscriptionName},
	})
	if err == nil {
		consumer.Ack(msg)
	} else {
		logger.ErrorContext(logCtx, "[pulsar] consumer error:"+err.Error())
		retryTimes := uint64(0)
		retryTimes = min(opts.RetryTimes, MAX_RETRY_TIMES)
		ACKMode := uint32(opts.ACKMode)
		rt := uint64(msg.RedeliveryCount())
		if ACKMode == 1 && rt < retryTimes {
			logger.InfofContext(logCtx, "[pulsar]consummer error and retry=> subscriptionName:"+cm.Subscription()+",initRetryTimes:%d,retryTimes:%d,ack:%d", retryTimes, rt, ACKMode)
			consumer.Nack(msg)
		} else {
			logger.InfofContext(logCtx, "[pulsar]consummer error and can not retry=> subscriptionName:"+cm.Subscription()+",initRetryTimes:%d,retryTimes:%d,ack:%d", retryTimes, rt, ACKMode)
			consumer.Ack(msg)
		}

	}
}

func (pc *PulsarClient) Close() {
	pc.client.Close()
}
