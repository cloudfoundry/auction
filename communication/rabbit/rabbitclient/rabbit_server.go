package rabbitclient

import (
	"sync"

	"github.com/streadway/amqp"
)

type Callback func([]byte) []byte

type RabbitServerInterface interface {
	ConnectAndEstablish() error
	Disconnect() error

	Handle(subject string, callback Callback)
}

type RabbitServer struct {
	id         string
	url        string
	connection *amqp.Connection
	channel    *amqp.Channel
	handlers   map[string]Callback
	lock       *sync.Mutex
}

func NewServer(id string, url string) RabbitServerInterface {
	return &RabbitServer{
		id:       id,
		url:      url,
		handlers: map[string]Callback{},
		lock:     &sync.Mutex{},
	}
}

func (r *RabbitServer) queueName() string {
	return r.id
}

func (r *RabbitServer) ConnectAndEstablish() error {
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

func (r *RabbitServer) Handle(subject string, callback Callback) {
	r.lock.Lock()
	r.handlers[subject] = callback
	r.lock.Unlock()
}

func (r *RabbitServer) dispatch(delivery amqp.Delivery) {
	r.lock.Lock()
	callback, ok := r.handlers[delivery.Type]
	r.lock.Unlock()
	if !ok {
		return
	}

	response := callback(delivery.Body)

	r.channel.Publish("", delivery.ReplyTo, false, false, amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: delivery.CorrelationId,
		Body:          response,
	})
}

func (r *RabbitServer) Disconnect() error {
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
