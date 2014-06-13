package repnatsclient

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/auction/communication/nats/natsmuxer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var RequestFailedError = errors.New("request failed")

type RepNatsClient struct {
	client     *natsmuxer.NATSMuxerClient
	timeout    time.Duration
	runTimeout time.Duration
	logger     *gosteno.Logger
}

func New(natsClient yagnats.NATSClient, timeout time.Duration, runTimeout time.Duration, logger *gosteno.Logger) (*RepNatsClient, error) {
	client := natsmuxer.NewNATSMuxerClient(natsClient)
	err := client.ListenForResponses()
	if err != nil {
		return nil, err
	}

	return &RepNatsClient{
		client:     client,
		timeout:    timeout,
		runTimeout: runTimeout,
		logger:     logger,
	}, nil
}

func (rep *RepNatsClient) BidForStartAuction(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
	}, "rep-nats-client.start-bid.fetching")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).BidForStartAuction)
	}
	payload, _ := json.Marshal(startAuctionInfo)

	responses, _ := rep.aggregateWithTimeout(subjects, payload, rep.timeout)

	results := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"malformed-payload": string(response),
				"error":             err.Error(),
			}, "rep-nats-client.start-bid.parse-failed")
			continue
		}
		results = append(results, bid)
	}

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
		"num-bids-received":  len(results),
	}, "rep-nats-client.start-bid.fetched")

	return results
}

func (rep *RepNatsClient) BidForStopAuction(repGuids []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	rep.logger.Infod(map[string]interface{}{
		"stop-auction-info": stopAuctionInfo,
		"num-rep-guids":     len(repGuids),
	}, "rep-nats-client.stop-bid.fetching")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).BidForStopAuction)
	}
	payload, _ := json.Marshal(stopAuctionInfo)

	responses, _ := rep.aggregateWithTimeout(subjects, payload, rep.timeout)

	results := auctiontypes.StopAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StopAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"malformed-payload": string(response),
				"error":             err.Error(),
			}, "rep-nats-client.stop-bid.parse-failed")
			continue
		}
		results = append(results, bid)
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

	subjects := []string{}
	subjectToRepGuid := map[string]string{}
	for _, repGuid := range repGuids {
		subject := nats.NewSubjects(repGuid).RebidThenTentativelyReserve
		subjects = append(subjects, subject)
		subjectToRepGuid[subject] = repGuid
	}
	payload, _ := json.Marshal(startAuctionInfo)

	responses, failedSubjects := rep.aggregateWithTimeout(subjects, payload, rep.timeout)

	results := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			rep.logger.Infod(map[string]interface{}{
				"malformed-payload": string(response),
				"error":             err.Error(),
			}, "rep-nats-client.bid-then-tentatively-reserve.parse-failed")
			continue
		}
		results = append(results, bid)
	}

	if len(failedSubjects) > 0 {
		releaseGuids := []string{}
		for _, failedSubject := range failedSubjects {
			releaseGuids = append(releaseGuids, subjectToRepGuid[failedSubject])
		}

		rep.ReleaseReservation(releaseGuids, startAuctionInfo)
	}

	rep.logger.Infod(map[string]interface{}{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
		"num-bids-received":  len(results),
	}, "rep-nats-client.bid-then-tentatively-reserve.fetched")

	return results
}

func (rep *RepNatsClient) ReleaseReservation(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	rep.logger.Infod(map[string]interface{}{
		"start-auction-info":   startAuctionInfo,
		"rep-guids-to-release": repGuids,
	}, "rep-nats-client.release-reservation.starting")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).ReleaseReservation)
	}
	payload, _ := json.Marshal(startAuctionInfo)

	rep.aggregateWithTimeout(subjects, payload, rep.timeout)

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
	payload, _ := json.Marshal(startAuction)
	_, err := rep.publishWithTimeout(subjects.Run, payload, rep.runTimeout)

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
	payload, _ := json.Marshal(instanceGuid)

	_, err := rep.publishWithTimeout(subjects.Stop, payload, rep.timeout)

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

func (rep *RepNatsClient) publishWithTimeout(subject string, payload []byte, timeout time.Duration) ([]byte, error) {
	response, err := rep.client.Request(subject, payload, timeout)
	if err != nil {
		return nil, err
	}

	if string(response) == "error" {
		return nil, RequestFailedError
	}

	return response, nil
}

func (rep *RepNatsClient) aggregateWithTimeout(subjects []string, payload []byte, timeout time.Duration) ([][]byte, []string) {
	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(subjects))

	lock := &sync.Mutex{}
	results := [][]byte{}
	failed := []string{}

	for _, subject := range subjects {
		go func(subject string) {
			defer allReceived.Done()

			result, err := rep.publishWithTimeout(subject, payload, timeout)
			if err != nil {
				rep.logger.Infod(map[string]interface{}{
					"error": err.Error(),
				}, "rep-nats-client.request-failed")

				lock.Lock()
				failed = append(failed, subject)
				lock.Unlock()

				return
			}

			lock.Lock()
			results = append(results, result)
			lock.Unlock()
		}(subject)
	}

	allReceived.Wait()

	return results, failed
}

//SIMULATION ONLY METHODS:

func (rep *RepNatsClient) TotalResources(repGuid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	subjects := nats.NewSubjects(repGuid)
	response, err := rep.publishWithTimeout(subjects.TotalResources, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	err = json.Unmarshal(response, &totalResources)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return totalResources
}

func (rep *RepNatsClient) SimulatedInstances(repGuid string) []auctiontypes.SimulatedInstance {
	var instances []auctiontypes.SimulatedInstance
	subjects := nats.NewSubjects(repGuid)
	response, err := rep.publishWithTimeout(subjects.SimulatedInstances, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	err = json.Unmarshal(response, &instances)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}

	return instances
}

func (rep *RepNatsClient) Reset(repGuid string) {
	subjects := nats.NewSubjects(repGuid)
	_, err := rep.publishWithTimeout(subjects.Reset, nil, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}

func (rep *RepNatsClient) SetSimulatedInstances(repGuid string, instances []auctiontypes.SimulatedInstance) {
	subjects := nats.NewSubjects(repGuid)
	payload, _ := json.Marshal(instances)
	_, err := rep.publishWithTimeout(subjects.SetSimulatedInstances, payload, rep.timeout)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}
