package gnats

import (
	"errors"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/skirrund/gcloud/bootstrap/env"
	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/mq"
	"github.com/skirrund/gcloud/tracer"
	"github.com/skirrund/gcloud/utils"
	"github.com/skirrund/gcloud/utils/worker"
)

const (
	SERVER_URL_KEY        = "nats.url"
	TIMEOUT_KEY           = "nats.timeout"
	USER_KEY              = "nats.user"
	PWD_KEY               = "nats.password"
	ServerName            = "server.name"
	MAX_RETRY_TIMES       = 50
	DefaultPullBatchSize  = 34
	DefaultStreamPrefix   = "public"
	NatsScheduleTarget    = "Nats-Schedule-Target"
	NatsSchedule          = "Nats-Schedule"
	ScheduleAt            = "@at "
	ScheduleSubjectSubfix = ".schedules."
)

type natsConn struct {
	conn    *nats.Conn
	js      nats.JetStreamContext
	appName string
}

var conn *natsConn

func NewDefaultConnection() (mq.IClient, error) {
	opts := Option{}
	cfg := env.GetInstance()
	utils.NewOptions(cfg, &opts)
	if len(opts.AppName) == 0 {
		opts.AppName = cfg.GetString(ServerName)
	}
	return NewConnection(opts)
}

func NewConnection(opt Option) (mq.IClient, error) {
	if conn != nil {
		return conn, nil
	}
	nc, err := connect(opt)
	if err != nil {
		return conn, err
	}
	js, err := nc.JetStream()
	if err != nil {
		return conn, err
	}
	conn = &natsConn{
		conn:    nc,
		appName: opt.AppName,
		js:      js,
	}
	return conn, err
}

func connect(opt Option) (*nats.Conn, error) {
	var opts []nats.Option
	if opt.Timeout > 0 {
		opts = append(opts, nats.Timeout(opt.Timeout))
	}
	if len(opt.AppName) > 0 {
		opts = append(opts, nats.Name(opt.AppName))
	}
	if len(opt.User) > 0 {
		opts = append(opts, nats.UserInfo(opt.User, opt.Password))
	}
	if opt.MaxPingsOutstanding > 0 {
		opts = append(opts, nats.MaxPingsOutstanding(opt.MaxPingsOutstanding))
	}
	url := opt.Url
	logger.Infof("[nats]start init nas natsConn:" + url)
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nc, err
	}
	logger.Infof("[nats]finished init natsConn")
	return nc, err
}

func (nc *natsConn) Send(msg *mq.Message) error {
	return nc.doSendSync(msg)
}

func (nc *natsConn) SendAsync(msg *mq.Message) error {
	return nc.doSendAsync(msg)
}

func (nc *natsConn) doSendSync(msg *mq.Message) error {
	nm, opts := createMsg(msg)
	if _, err := nc.js.PublishMsg(nm, opts...); err != nil {
		return err
	}
	return nil
}

func (nc *natsConn) doSendAsync(msg *mq.Message) error {
	nm, opts := createMsg(msg)
	if _, err := nc.js.PublishMsgAsync(nm, opts...); err != nil {
		return err
	}
	return nil
}

func getScheduleSubject(stream string, deliverAt time.Time, deliverAfter time.Duration) (subject, schedule string) {
	subject = stream + ScheduleSubjectSubfix + utils.Uuid()
	if deliverAt.Compare(time.Now()) > 0 {
		schedule = ScheduleAt + deliverAt.Format(time.RFC3339)
	} else if deliverAfter > 0 {
		schedule = ScheduleAt + time.Now().Add(deliverAfter).Format(time.RFC3339)
	}
	return
}

func createMsg(msg *mq.Message) (natsMsg *nats.Msg, opts []nats.PubOpt) {
	subject := strings.ReplaceAll(msg.Topic, "/", "-")
	stream := msg.NatsOpts.Stream
	if len(stream) == 0 {
		stream = subject
	}
	opts = append(opts, nats.ExpectStream(stream))
	msg.Topic = subject
	header := make(nats.Header)
	natsMsg = &nats.Msg{
		Data:    msg.Payload,
		Subject: subject,
	}
	if msg.DeliverAfter > 0 || msg.DeliverAt.Compare(time.Now()) > 0 {
		scheduleSubject, schedule := getScheduleSubject(stream, msg.DeliverAt, msg.DeliverAfter)
		header.Set(NatsSchedule, schedule)
		header.Set(NatsScheduleTarget, subject)
		natsMsg.Subject = scheduleSubject
	}
	if len(msg.Header) > 0 {
		for k, v := range msg.Header {
			header.Set(k, v)
		}
	}
	natsMsg.Header = header
	return
}

func (nc *natsConn) Subscribe(opts mq.ConsumerOptions) error {
	go nc.doSubscribe(opts)
	return nil
}

func (nc *natsConn) SubscribeSync(opts mq.ConsumerOptions) error {
	return nc.doSubscribe(opts)
}

func doPanic(canPanic bool, err error) {
	if canPanic && err != nil {
		panic(err)
	}
}

func (nc *natsConn) doSubscribe(opts mq.ConsumerOptions) error {
	subject := strings.ReplaceAll(opts.Topic, "/", "-")
	stream := opts.NatsOpts.Stream
	if len(stream) == 0 {
		stream = subject
	}
	opts.Topic = subject
	opts.NatsOpts.Stream = stream
	subscriptionName := strings.ReplaceAll(opts.SubscriptionName, "/", "-")
	opts.SubscriptionName = subscriptionName
	logger.Infof("[nats]ConsumerOptions:%+v", opts)
	if opts.RetryTimes == 0 {
		opts.RetryTimes = MAX_RETRY_TIMES
	}
	channelSize := opts.MaxMessageChannelSize
	if channelSize == 0 {
		channelSize = 200
	}
	opts.MaxMessageChannelSize = channelSize
	pullBatchSize := opts.NatsOpts.PullBatchSize
	if pullBatchSize <= 0 {
		pullBatchSize = DefaultPullBatchSize
	}
	opts.NatsOpts.PullBatchSize = pullBatchSize
	info, err := nc.js.ConsumerInfo(stream, subscriptionName)
	if err != nil {
		doPanic(opts.IsErrorPanic, err)
		return err
	}
	if info != nil {
		cfg := info.Config
		fs := cfg.FilterSubject
		if len(fs) == 0 || fs != subject {
			err := errors.New("nats FilterSubject not matched:stream=>" + opts.NatsOpts.Stream + ",FilterSubject=>" + fs + ",topic=>" + subject + ",consumer=>" + subscriptionName)
			doPanic(opts.IsErrorPanic, err)
			return err
			// cfg.FilterSubject = topic
			// cfg.FilterSubjects = []string{}
			// _, err := nc.js.UpdateConsumer(opts.NatsOpts.Stream, &cfg)
			// if err != nil {
			// 	doPanic(opts.IsErrorPanic, err)
			// 	return err
			// }
		}
		//pull
		if len(cfg.DeliverGroup) == 0 {
			numcpu := min(runtime.NumCPU(), 4)
			for range numcpu - 1 {
				go nc.pullSubscribe(opts, info.Config)
			}
			return nc.pullSubscribe(opts, info.Config)
		} else {
			return nc.pushSubscribe(opts, info.Config)
		}
	} else {
		errMsg := "nats get consumer nil:" + opts.NatsOpts.Stream + "=>" + opts.SubscriptionName
		logger.Error(errMsg)
		err = errors.New(errMsg)
		doPanic(opts.IsErrorPanic, err)
		return err
	}
}
func (nc *natsConn) pushSubscribe(opts mq.ConsumerOptions, cfg nats.ConsumerConfig) error {
	subscriptionName := opts.SubscriptionName
	topic := opts.Topic
	channelSize := opts.MaxMessageChannelSize
	if len(cfg.DeliverGroup) == 0 || cfg.DeliverGroup != subscriptionName {
		err := errors.New("nats DeliverGroup not matched")
		doPanic(opts.IsErrorPanic, err)
		return errors.New("nats DeliverGroup not matched")
	}
	msgsChan := make(chan *nats.Msg, channelSize)
	_, err := nc.js.ChanQueueSubscribe(topic, subscriptionName, msgsChan, nats.Bind(opts.NatsOpts.Stream, subscriptionName), nats.ManualAck())
	if err != nil {
		doPanic(opts.IsErrorPanic, err)
		logger.Error("[nats]ChanQueueSubscribe error:", err.Error())
		return err
	}
	for msg := range msgsChan {
		go consume(msg, opts)
	}
	return nil
}
func (nc *natsConn) pullSubscribe(opts mq.ConsumerOptions, cfg nats.ConsumerConfig) error {
	subscriptionName := opts.SubscriptionName
	topic := opts.Topic
	sub, err := nc.js.PullSubscribe(topic, subscriptionName, nats.Bind(opts.NatsOpts.Stream, subscriptionName), nats.ManualAck())
	if err != nil {
		logger.Error("[nats]ChanQueueSubscribe error:", err.Error())
		doPanic(opts.IsErrorPanic, err)
		return err
	}
	pullBatchSize := opts.NatsOpts.PullBatchSize
	execWorker := worker.Init(pullBatchSize * 2)
	for {
		batch, _ := sub.FetchBatch(pullBatchSize)
		for msg := range batch.Messages() {
			execWorker.Execute(func() {
				consume(msg, opts)
			})
		}
	}
}
func consume(msg *nats.Msg, opts mq.ConsumerOptions) {
	logCtx := tracer.NewTraceIDContext()
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorContext(logCtx, "[nats] consumer panic recover :", err, "\n", string(debug.Stack()))
		}
	}()
	metaData, err := msg.Metadata()
	if err != nil {
		logger.ErrorContext(logCtx, "[nats] consumer Metadata error :", err.Error())
	}
	data := msg.Data
	msgStr := string(data)
	logger.InfoContext(logCtx, "[nats] consumer msg:", msgStr)
	redeliveryCount := metaData.NumDelivered
	logger.InfofContext(logCtx, "[nats] consumer info=>subName:%s,reDeliveryCount:%d,publishTime:%s,topic:%s", opts.SubscriptionName, metaData.NumDelivered, metaData.Timestamp.Format(time.DateTime), msg.Subject)
	err = opts.MessageListener(logCtx, &mq.Message{
		Payload:         data,
		RedeliveryCount: redeliveryCount,
		SubOpts:         mq.SubOpts{Name: opts.SubscriptionName},
	})
	if err == nil {
		msg.Ack()
	} else {
		logger.ErrorContext(logCtx, "[nats] consumer error:"+err.Error())
		retryTimes := uint64(0)
		retryTimes = min(opts.RetryTimes, MAX_RETRY_TIMES)
		ackMode := opts.ACKMode
		if ackMode == mq.ACK_MANUAL && redeliveryCount < retryTimes {
			msg.NakWithDelay(time.Duration(2*redeliveryCount-1) * time.Second)
			logger.InfofContext(logCtx, "[nats]consummer error and retry=> subscriptionName:"+opts.SubscriptionName+",initRetryTimes:%d,retryTimes:%d,ack:%d", retryTimes, redeliveryCount, ackMode)
		} else {
			msg.Ack()
			logger.InfofContext(logCtx, "[nats]consummer error and can not retry=> subscriptionName:"+opts.SubscriptionName+",initRetryTimes:%d,retryTimes:%d,ack:%d", retryTimes, redeliveryCount, ackMode)
		}

	}
}

func (nc *natsConn) Close() {
	nc.conn.Drain()
}
