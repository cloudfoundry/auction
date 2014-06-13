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

func (rep *RepNatsClient) BidForStartAuction(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
	}, "rep-nats-client.bid.fetching")

	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(repGuids))
	responses := make(chan auctiontypes.StartAuctionBid, len(repGuids))

	n := 0
	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		n++
		defer allReceived.Done()
		var result auctiontypes.StartAuctionBid
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"unparseable-message": msg.Payload,
				"error":               err.Error(),
			}, "rep-nats-client.bid.failed-to-parse-message")
			return
		}

		responses <- result
	})

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error": err.Error(),
		}, "rep-nats-client.bid.failed-to-fetch")
		return auctiontypes.StartAuctionBids{}
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(startAuctionInfo)

	for _, repGuid := range repGuids {
		subjects := nats.NewSubjects(repGuid)
		rep.client.PublishWithReplyTo(subjects.BidForStartAuction, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		rep.logger.Info("rep-nats-client.bid.did-not-receive-all-bids")
	}

	results := auctiontypes.StartAuctionBids{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
		"num-bids-received":  len(results),
	}, "rep-nats-client.bid.fetched")

	return results
}

func (rep *RepNatsClient) BidForStopAuction(repGuids []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	rep.logger.Infod(map[string]interface{}{
		"stop-auction-info": stopAuctionInfo,
		"num-rep-guids":     len(repGuids),
	}, "rep-nats-client.stop-bid.fetching")

	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(repGuids))
	responses := make(chan auctiontypes.StopAuctionBid, len(repGuids))

	n := 0
	subscriptionID, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		n++
		defer allReceived.Done()
		var result auctiontypes.StopAuctionBid
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"unparseable-message": msg.Payload,
				"error":               err.Error(),
			}, "rep-nats-client.stop-bid.failed-to-parse-message")
			return
		}

		responses <- result
	})

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error": err.Error(),
		}, "rep-nats-client.stop-bid.failed-to-fetch")
		return []auctiontypes.StopAuctionBid{}
	}

	defer rep.client.Unsubscribe(subscriptionID)

	payload, _ := json.Marshal(stopAuctionInfo)

	for _, repGuid := range repGuids {
		subjects := nats.NewSubjects(repGuid)
		rep.client.PublishWithReplyTo(subjects.BidForStopAuction, replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
		rep.logger.Info("rep-nats-client.stop-bid.did-not-receive-all-bids")
	}

	results := auctiontypes.StopAuctionBids{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	rep.logger.Infod(map[string]interface{}{
		"stop-auction-info": stopAuctionInfo,
		"num-rep-guids":     len(repGuids),
		"num-bids-received": len(results),
	}, "rep-nats-client.stop-bid.fetched")

	return results
}

func (rep *RepNatsClient) RebidThenTentativelyReserve(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
	}, "rep-nats-client.bid-then-tentatively-reserve.starting")

	resultChan := make(chan auctiontypes.StartAuctionBid, 0)
	for _, repGuid := range repGuids {
		go func(repGuid string) {
			result := auctiontypes.StartAuctionBid{}
			subjects := nats.NewSubjects(repGuid)
			err := rep.publishWithTimeout(subjects.RebidThenTentativelyReserve, startAuctionInfo, &result, rep.timeout)
			if err != nil {
				rep.logger.Infod(map[string]interface{}{
					"error":    err.Error(),
					"rep-guid": repGuid,
				}, "rep-nats-client.bid-then-tentatively-reserve.failed")

				result = auctiontypes.StartAuctionBid{Error: err.Error()}
				rep.publishWithTimeout(subjects.ReleaseReservation, startAuctionInfo, nil, rep.timeout)
			}
			resultChan <- result
		}(repGuid)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range repGuids {
		results = append(results, <-resultChan)
	}

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
		"num-bids-received":  len(results),
	}, "rep-nats-client.bid-then-tentatively-reserve.done")

	return results
}

func (rep *RepNatsClient) ReleaseReservation(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info":   startAuctionInfo,
		"rep-guids-to-release": repGuids,
	}, "rep-nats-client.release-reservation.starting")

	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(repGuids))

	for _, repGuid := range repGuids {
		go func(repGuid string) {
			subjects := nats.NewSubjects(repGuid)
			err := rep.publishWithTimeout(subjects.ReleaseReservation, startAuctionInfo, nil, rep.timeout)
			if err != nil {
				rep.logger.Infod(map[string]interface{}{
					"error":    err.Error(),
					"rep-guid": repGuid,
				}, "rep-nats-client.release-reservation.failed")
			}
			allReceived.Done()
		}(repGuid)
	}

	allReceived.Wait()

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info":   startAuctionInfo,
		"rep-guids-to-release": repGuids,
	}, "rep-nats-client.release-reservation.done")
}

func (rep *RepNatsClient) Run(repGuid string, startAuction models.LRPStartAuction) {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuction,
		"rep-guid":           repGuid,
	}, "rep-nats-client.run.starting")

	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.Run, startAuction, nil, rep.runTimeout)

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error":              err.Error(),
			"start-auction-info": startAuction,
			"rep-guid":           repGuid,
		}, "rep-nats-client.run.failed")
	}

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuction,
		"rep-guid":           repGuid,
	}, "rep-nats-client.run.done")
}

func (rep *RepNatsClient) Stop(repGuid string, instanceGuid string) {
	rep.logger.Infod(map[string]interface{}{
		"instance-guid": instanceGuid,
		"rep-guid":      repGuid,
	}, "rep-nats-client.stop.starting")

	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.Stop, instanceGuid, nil, rep.timeout)

	if err != nil {
		rep.logger.Errord(map[string]interface{}{
			"error":         err.Error(),
			"instance-guid": instanceGuid,
			"rep-guid":      repGuid,
		}, "rep-nats-client.stop.failed")
	}

	rep.logger.Infod(map[string]interface{}{
		"instance-guid": instanceGuid,
		"rep-guid":      repGuid,
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

func (rep *RepNatsClient) TotalResources(repGuid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.TotalResources, nil, &totalResources, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return totalResources
}

func (rep *RepNatsClient) SimulatedInstances(repGuid string) []auctiontypes.SimulatedInstance {
	var instances []auctiontypes.SimulatedInstance
	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.SimulatedInstances, nil, &instances, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return instances
}

func (rep *RepNatsClient) Reset(repGuid string) {
	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.Reset, nil, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}

func (rep *RepNatsClient) SetSimulatedInstances(repGuid string, instances []auctiontypes.SimulatedInstance) {
	subjects := nats.NewSubjects(repGuid)
	err := rep.publishWithTimeout(subjects.SetSimulatedInstances, instances, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}
