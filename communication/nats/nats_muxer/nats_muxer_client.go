package nats_muxer

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry/yagnats"
)

var TimeoutError = errors.New("timeout")

type NATSMuxerClient struct {
	client         yagnats.NATSClient
	replyGuid      string
	subscriptionID int64
	correlationID  int64
	requests       map[int64]chan []byte
	lock           *sync.Mutex
}

type message struct {
	CorrelationID int64
	Payload       []byte
}

func NewNATSMuxerClient(client yagnats.NATSClient) *NATSMuxerClient {
	replyGuid := util.RandomGuid()
	return &NATSMuxerClient{
		client:        client,
		replyGuid:     replyGuid,
		correlationID: 0,
		lock:          &sync.Mutex{},
		requests:      map[int64]chan []byte{},
	}
}

func (c *NATSMuxerClient) ListenForResponses() error {
	subscriptionID, err := c.client.Subscribe(c.replyGuid, func(msg *yagnats.Message) {
		go c.handleResponse(msg)
	})

	if err != nil {
		return err
	}

	c.subscriptionID = subscriptionID
	return nil
}

func (c *NATSMuxerClient) Shutdown() error {
	return c.client.Unsubscribe(c.subscriptionID)
}

func (c *NATSMuxerClient) Request(subject string, payload []byte, timeout time.Duration) ([]byte, error) {
	response := make(chan []byte, 0)
	correlationID := atomic.AddInt64(&c.correlationID, 1)

	c.lock.Lock()
	c.requests[correlationID] = response
	c.lock.Unlock()

	defer func() {
		c.lock.Lock()
		delete(c.requests, correlationID)
		c.lock.Unlock()
	}()

	msg := message{
		CorrelationID: correlationID,
		Payload:       payload,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	err = c.client.PublishWithReplyTo(subject, c.replyGuid, payload)
	if err != nil {
		return nil, err
	}

	select {
	case payload := <-response:
		return payload, nil
	case <-time.After(timeout):
		return nil, TimeoutError
	}

	panic("can't get here")
}

func (c *NATSMuxerClient) handleResponse(msg *yagnats.Message) {
	response := message{}
	err := json.Unmarshal(msg.Payload, &response)
	if err != nil {
		return
	}

	c.lock.Lock()
	responseChan, ok := c.requests[response.CorrelationID]
	c.lock.Unlock()
	if !ok {
		return
	}

	responseChan <- response.Payload
}
