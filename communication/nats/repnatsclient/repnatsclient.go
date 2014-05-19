package repnatsclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry/yagnats"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepNatsClient struct {
	client  yagnats.NATSClient
	timeout time.Duration
}

func New(client yagnats.NATSClient, timeout time.Duration) *RepNatsClient {
	return &RepNatsClient{
		client:  client,
		timeout: timeout,
	}
}

func (rep *RepNatsClient) publishWithTimeout(guid string, subject string, req interface{}, resp interface{}) (err error) {
	replyTo := util.RandomGuid()
	c := make(chan []byte, 1)

	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		c <- msg.Payload
	})
	if err != nil {
		return err
	}
	defer rep.client.Unsubscribe(subscriptionID)

	payload := []byte{}
	if req != nil {
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	rep.client.PublishWithReplyTo(guid+"."+subject, replyTo, payload)

	select {
	case payload := <-c:
		if string(payload) == "error" {
			return RequestFailedError
		}

		if resp != nil {
			return json.Unmarshal(payload, resp)
		}

		return nil

	case <-time.After(rep.timeout):
		return TimeoutError
	}
}

func (rep *RepNatsClient) TotalResources(guid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	err := rep.publishWithTimeout(guid, "total_resources", nil, &totalResources)
	if err != nil {
		panic(err)
	}

	return totalResources
}

func (rep *RepNatsClient) Instances(guid string) []auctiontypes.Instance {
	var instances []auctiontypes.Instance
	err := rep.publishWithTimeout(guid, "instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepNatsClient) Reset(guid string) {
	err := rep.publishWithTimeout(guid, "reset", nil, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepNatsClient) SetInstances(guid string, instances []auctiontypes.Instance) {
	err := rep.publishWithTimeout(guid, "set_instances", instances, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepNatsClient) batch(subject string, guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))
	responses := make(chan auctiontypes.ScoreResult, len(guids))

	n := 0
	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		n++
		defer allReceived.Done()
		var result auctiontypes.ScoreResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			return
		}

		responses <- result
	})

	if err != nil {
		return auctiontypes.ScoreResults{}
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(instance)

	for _, guid := range guids {
		rep.client.PublishWithReplyTo(guid+"."+subject, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		println("TIMING OUT!!")
	}

	results := auctiontypes.ScoreResults{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	return results
}

func (rep *RepNatsClient) Score(guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	return rep.batch("score", guids, instance)
}

func (rep *RepNatsClient) ScoreThenTentativelyReserve(guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	return rep.batch("score_then_tentatively_reserve", guids, instance)
}

func (rep *RepNatsClient) ReleaseReservation(guids []string, instance auctiontypes.Instance) {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))

	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		allReceived.Done()
	})

	if err != nil {
		return
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(instance)

	for _, guid := range guids {
		rep.client.PublishWithReplyTo(guid+".release-reservation", replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		println("TIMING OUT!!")
	}
}

func (rep *RepNatsClient) Claim(guid string, instance auctiontypes.Instance) {
	err := rep.publishWithTimeout(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
