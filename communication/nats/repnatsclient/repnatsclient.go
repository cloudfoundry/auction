package repnatsclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/yagnats"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepNatsClient struct {
	client     yagnats.NATSClient
	timeout    time.Duration
	runTimeout time.Duration
}

func New(client yagnats.NATSClient, timeout time.Duration, runTimeout time.Duration) *RepNatsClient {
	return &RepNatsClient{
		client:     client,
		timeout:    timeout,
		runTimeout: runTimeout,
	}
}

func (rep *RepNatsClient) publishWithTimeout(subject string, req interface{}, resp interface{}, timeout time.Duration) (err error) {
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

	rep.client.PublishWithReplyTo(subject, replyTo, payload)

	select {
	case payload := <-c:
		if string(payload) == "error" {
			return RequestFailedError
		}

		if resp != nil {
			return json.Unmarshal(payload, resp)
		}

		return nil

	case <-time.After(timeout):
		return TimeoutError
	}
}

func (rep *RepNatsClient) Score(guids []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
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
		rep.client.PublishWithReplyTo(guid+".score", replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		log.Println("timed out fetching scores")
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

func (rep *RepNatsClient) ScoreThenTentativelyReserve(guids []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
	resultChan := make(chan auctiontypes.ScoreResult, 0)
	for _, guid := range guids {
		go func(guid string) {
			result := auctiontypes.ScoreResult{}
			subjects := nats.NewSubjects(guid)
			err := rep.publishWithTimeout(subjects.ScoreThenTentativelyReserve, instance, &result, rep.timeout)
			if err != nil {
				log.Println("errored getting a reservation:", err.Error())
				result = auctiontypes.ScoreResult{Error: err.Error()}
				rep.publishWithTimeout(subjects.ReleaseReservation, instance, nil, rep.timeout)
			}
			resultChan <- result
		}(guid)
	}

	results := auctiontypes.ScoreResults{}
	for _ = range guids {
		results = append(results, <-resultChan)
	}

	return results
}

func (rep *RepNatsClient) ReleaseReservation(guids []string, instance auctiontypes.LRPAuctionInfo) {
	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))

	for _, guid := range guids {
		go func(guid string) {
			subjects := nats.NewSubjects(guid)
			err := rep.publishWithTimeout(subjects.ReleaseReservation, instance, nil, rep.timeout)
			if err != nil {
				log.Println("errored releasing a rservation:", err.Error())
			}
			allReceived.Done()
		}(guid)
	}

	allReceived.Wait()
}

func (rep *RepNatsClient) Run(guid string, instance models.LRPStartAuction) {
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.Run, instance, nil, rep.runTimeout)
	if err != nil {
		log.Println("failed to run:", err.Error())
	}
}

//SIMULATION ONLY METHODS:

func (rep *RepNatsClient) TotalResources(guid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.TotalResources, nil, &totalResources, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return totalResources
}

func (rep *RepNatsClient) LRPAuctionInfos(guid string) []auctiontypes.LRPAuctionInfo {
	var instances []auctiontypes.LRPAuctionInfo
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.LrpAuctionInfos, nil, &instances, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return instances
}

func (rep *RepNatsClient) Reset(guid string) {
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.Reset, nil, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}

func (rep *RepNatsClient) SetLRPAuctionInfos(guid string, instances []auctiontypes.LRPAuctionInfo) {
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.SetLrpAuctionInfos, instances, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}
