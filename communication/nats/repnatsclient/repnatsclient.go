package repnatsclient

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepNatsClient struct {
	client     yagnats.NATSClient
	timeout    time.Duration
	runTimeout time.Duration
	logger     *gosteno.Logger
}

func New(client yagnats.NATSClient, timeout time.Duration, runTimeout time.Duration, logger *gosteno.Logger) *RepNatsClient {
	return &RepNatsClient{
		client:     client,
		timeout:    timeout,
		runTimeout: runTimeout,
		logger:     logger,
	}
}

func (rep *RepNatsClient) Score(guids []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
	rep.logger.Infod(map[string]interface{}{
		"auction-info":  instance,
		"num-rep-guids": len(guids),
	}, "rep-nats-client.score.fetching")

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
			rep.logger.Infod(map[string]interface{}{
				"unparseable-message": msg.Payload,
				"error":               err.Error(),
			}, "rep-nats-client.score.failed-to-parse-message")
			return
		}

		responses <- result
	})

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error": err.Error(),
		}, "rep-nats-client.score.failed-to-fetch")
		return auctiontypes.ScoreResults{}
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(instance)

	for _, guid := range guids {
		subjects := nats.NewSubjects(guid)
		rep.client.PublishWithReplyTo(subjects.Score, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		rep.logger.Info("rep-nats-client.score.did-not-receive-all-scores")
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

	rep.logger.Infod(map[string]interface{}{
		"auction-info":        instance,
		"num-rep-guids":       len(guids),
		"num-scores-received": len(results),
	}, "rep-nats-client.score.fetched")

	return results
}

func (rep *RepNatsClient) StopScore(guids []string, stopAuctionInfo auctiontypes.LRPStopAuctionInfo) auctiontypes.StopScoreResults {
	rep.logger.Infod(map[string]interface{}{
		"stop-auction-info": stopAuctionInfo,
		"num-rep-guids":     len(guids),
	}, "rep-nats-client.stop-score.fetching")

	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))
	responses := make(chan auctiontypes.StopScoreResult, len(guids))

	n := 0
	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		n++
		defer allReceived.Done()
		var result auctiontypes.StopScoreResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"unparseable-message": msg.Payload,
				"error":               err.Error(),
			}, "rep-nats-client.stop-score.failed-to-parse-message")
			return
		}

		responses <- result
	})

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error": err.Error(),
		}, "rep-nats-client.stop-score.failed-to-fetch")
		return []auctiontypes.StopScoreResult{}
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(stopAuctionInfo)

	for _, guid := range guids {
		subjects := nats.NewSubjects(guid)
		rep.client.PublishWithReplyTo(subjects.StopScore, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		rep.logger.Info("rep-nats-client.stop-score.did-not-receive-all-scores")
	}

	results := auctiontypes.StopScoreResults{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	rep.logger.Infod(map[string]interface{}{
		"stop-auction-info":   stopAuctionInfo,
		"num-rep-guids":       len(guids),
		"num-scores-received": len(results),
	}, "rep-nats-client.stop-score.fetched")

	return results
}

func (rep *RepNatsClient) ScoreThenTentativelyReserve(guids []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
	rep.logger.Infod(map[string]interface{}{
		"auction-info":  instance,
		"num-rep-guids": len(guids),
	}, "rep-nats-client.score-then-tentatively-reserve.starting")

	resultChan := make(chan auctiontypes.ScoreResult, 0)
	for _, guid := range guids {
		go func(guid string) {
			result := auctiontypes.ScoreResult{}
			subjects := nats.NewSubjects(guid)
			err := rep.publishWithTimeout(subjects.ScoreThenTentativelyReserve, instance, &result, rep.timeout)
			if err != nil {
				rep.logger.Infod(map[string]interface{}{
					"error":    err.Error(),
					"rep-guid": guid,
				}, "rep-nats-client.score-then-tentatively-reserve.failed")

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

	rep.logger.Infod(map[string]interface{}{
		"auction-info":        instance,
		"num-rep-guids":       len(guids),
		"num-scores-received": len(results),
	}, "rep-nats-client.score-then-tentatively-reserve.done")

	return results
}

func (rep *RepNatsClient) ReleaseReservation(guids []string, instance auctiontypes.LRPAuctionInfo) {
	rep.logger.Infod(map[string]interface{}{
		"auction-info":         instance,
		"rep-guids-to-release": guids,
	}, "rep-nats-client.release-reservation.starting")

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))

	for _, guid := range guids {
		go func(guid string) {
			subjects := nats.NewSubjects(guid)
			err := rep.publishWithTimeout(subjects.ReleaseReservation, instance, nil, rep.timeout)
			if err != nil {
				rep.logger.Infod(map[string]interface{}{
					"error":    err.Error(),
					"rep-guid": guid,
				}, "rep-nats-client.release-reservation.failed")
			}
			allReceived.Done()
		}(guid)
	}

	allReceived.Wait()

	rep.logger.Infod(map[string]interface{}{
		"auction-info":         instance,
		"rep-guids-to-release": guids,
	}, "rep-nats-client.release-reservation.done")
}

func (rep *RepNatsClient) Run(guid string, instance models.LRPStartAuction) {
	rep.logger.Infod(map[string]interface{}{
		"auction-info": instance,
		"rep-guid":     guid,
	}, "rep-nats-client.run.starting")

	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.Run, instance, nil, rep.runTimeout)

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error":        err.Error(),
			"auction-info": instance,
			"rep-guid":     guid,
		}, "rep-nats-client.run.failed")
	}

	rep.logger.Infod(map[string]interface{}{
		"auction-info": instance,
		"rep-guid":     guid,
	}, "rep-nats-client.run.done")
}

func (rep *RepNatsClient) Stop(guid string, instanceGuid string) {
	rep.logger.Infod(map[string]interface{}{
		"instance-guid": instanceGuid,
		"rep-guid":      guid,
	}, "rep-nats-client.stop.starting")

	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.Stop, instanceGuid, nil, rep.timeout)

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error":         err.Error(),
			"instance-guid": instanceGuid,
			"rep-guid":      guid,
		}, "rep-nats-client.stop.failed")
	}

	rep.logger.Infod(map[string]interface{}{
		"instance-guid": instanceGuid,
		"rep-guid":      guid,
	}, "rep-nats-client.stop.done")
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

func (rep *RepNatsClient) SimulatedInstances(guid string) []auctiontypes.SimulatedInstance {
	var instances []auctiontypes.SimulatedInstance
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.SimulatedInstances, nil, &instances, rep.timeout)
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

func (rep *RepNatsClient) SetSimulatedInstances(guid string, instances []auctiontypes.SimulatedInstance) {
	subjects := nats.NewSubjects(guid)
	err := rep.publishWithTimeout(subjects.SetSimulatedInstances, instances, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}
