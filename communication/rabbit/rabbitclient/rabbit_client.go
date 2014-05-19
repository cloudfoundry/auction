package rabbitclient

import (
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/streadway/amqp"
)

var TimeoutError = errors.New("timeout")

type RabbitClientInterface interface {
	ConnectAndEstablish() error
	Disconnect() error

	Request(recipientID string, subject string, payload []byte, timeout time.Duration) ([]byte, error)
}

type RabbitClient struct {
	id         string
	url        string
	connection *amqp.Connection
	channel    *amqp.Channel
	requests   map[string]chan amqp.Delivery
	lock       *sync.Mutex
}

func NewClient(id string, url string) RabbitClientInterface {
	return &RabbitClient{
		id:       id,
		url:      url,
		requests: map[string]chan amqp.Delivery{},
		lock:     &sync.Mutex{},
	}
}

func (r *RabbitClient) queueName() string {
	return r.id + "-response"
}

func (r *RabbitClient) ConnectAndEstablish() error {
	var err error
	r.connection, err = amqp.Dial(r.url)
	if err != nil {
		return err
	}

	r.channel, err = r.connection.Channel()
	if err != nil {
		r.connection.Close()
		return err
	}

	_, err = r.channel.QueueDeclare(r.queueName(), false, true, false, false, nil)
	if err != nil {
		r.Disconnect()
		return err
	}

	deliveries, err := r.channel.Consume(r.queueName(), "", false, true, false, false, nil)
	if err != nil {
		r.Disconnect()
		return err
	}

	go func() {
		for delivery := range deliveries {
			delivery.Ack(false)
			r.dispatch(delivery)
		}
	}()

	return nil
}

func (r *RabbitClient) Request(recipientID string, subject string, payload []byte, timeout time.Duration) ([]byte, error) {
	c := make(chan amqp.Delivery, 1)
	guid := util.RandomGuid()
	r.lock.Lock()
	r.requests[guid] = c
	r.lock.Unlock()

	defer func() {
		r.lock.Lock()
		delete(r.requests, guid)
		r.lock.Unlock()
	}()

	err := r.channel.Publish("", recipientID, false, false, amqp.Publishing{
		ContentType:   "application/json",
		Type:          subject,
		ReplyTo:       r.queueName(),
		CorrelationId: guid,
		Body:          payload,
	})

	if err != nil {
		return []byte{}, err
	}

	select {
	case delivery := <-c:
		return delivery.Body, nil
	case <-time.After(timeout):
		return []byte{}, TimeoutError
	}
}

func (r *RabbitClient) dispatch(delivery amqp.Delivery) {
	guid := delivery.CorrelationId
	r.lock.Lock()
	c, ok := r.requests[guid]
	r.lock.Unlock()
	if !ok {
		return
	}

	c <- delivery
}

func (r *RabbitClient) Disconnect() error {
	chanErr := r.channel.Close()
	connErr := r.connection.Close()

	if chanErr != nil {
		return connErr
	}

	if connErr != nil {
		return connErr
	}

	return nil
}
