package pulsar

import (
	"context"
	"errors"

	//	"sync"

	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/mq"
	"github.com/skirrund/gcloud/tracer"

	"github.com/apache/pulsar-client-go/pulsar"
)

var producers = make(map[string]pulsar.Producer)

func (pc *PulsarClient) getProducer(topic string) (pulsar.Producer, error) {
	p, ok := producers[topic]
	if ok && p != nil {
		logger.Info("[pulsar]load producer fromcache1:", topic, ",", true)
		return p, nil
	}
	pc.mt.Lock()
	defer pc.mt.Unlock()
	p, ok = producers[topic]
	var err error
	if !ok || p == nil {
		p, err = createProducer(topic)
		producers[topic] = p
	} else {
		logger.Info("[pulsar]load producer fromcache2:", topic, ",", true)
	}

	return p, err
}

func createProducer(topic string) (pulsar.Producer, error) {
	logger.Info("[pulsar]start create pulsar.Producer:", topic)
	pp := pulsar.ProducerOptions{
		Topic:  topic,
		Name:   getAppName(pulsarClient.appName),
		Schema: pulsar.NewJSONSchema(`"string"`, nil),
	}

	producer, err := pulsarClient.client.CreateProducer(pp)
	if err != nil {
		logger.Error("[pulsar]error create pulsar.Producer:", err)
	} else {
		logger.Info("[pulsar]finished create pulsar.Producer:", topic)
	}
	return producer, err
}

func createMsg(msg *mq.Message) *pulsar.ProducerMessage {
	message := &pulsar.ProducerMessage{
		Value: string(msg.Payload),
	}
	if msg.DeliverAfter > 0 {
		message.DeliverAfter = msg.DeliverAfter
	}
	if !msg.DeliverAt.IsZero() {
		message.DeliverAt = msg.DeliverAt
	}
	return message
}
func (pc *PulsarClient) doSend(msg *mq.Message) error {
	topic := msg.Topic
	if len(topic) == 0 {
		return errors.New("[pulsar] topic is empty")
	}
	logCtx := tracer.NewTraceIDContext()
	logger.InfoContext(logCtx, "[pulsar] send msg =>topic:"+topic+":"+string(msg.Payload))
	message := createMsg(msg)
	producer, err := pc.getProducer(topic)
	if err != nil {
		return err
	}
	msgId, err := producer.Send(context.Background(), message)
	if err != nil {
		logger.InfoContext(logCtx, "[pulsar]发送消息失败: ", err)
		return err
	}
	if msgId == nil {
		return errors.New("[pulsar]发送消息失败[messageId为空]:" + topic)
	}
	return nil
}

func (pc *PulsarClient) doSendAsync(msg *mq.Message) error {
	var err error
	topic := msg.Topic
	if len(topic) == 0 {
		err = errors.New("[pulsar] topic is empty")
		logger.Error(err.Error())
		return err
	}
	message := createMsg(msg)
	p, err := pc.getProducer(topic)
	if err != nil {
		return err
	}
	p.SendAsync(context.Background(), message, func(msgId pulsar.MessageID, msg *pulsar.ProducerMessage, err error) {
		if err != nil {
			logger.Error("[pulsar]发送doSendAsync消息失败:", err)
		} else {
			logger.Info("[pulsar] doSendAsync finish:", msgId)
		}
	})
	return nil
}
