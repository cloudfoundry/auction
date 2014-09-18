package auction_nats_client

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/auction/communication/nats/nats_muxer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/yagnats"
	"github.com/pivotal-golang/lager"
)

var RequestFailedError = errors.New("request failed")

type AuctionNATSClient struct {
	client  *nats_muxer.NATSMuxerClient
	timeout time.Duration
	logger  lager.Logger
}

func New(natsClient yagnats.ApceraWrapperNATSClient, timeout time.Duration, logger lager.Logger) (*AuctionNATSClient, error) {
	client := nats_muxer.NewNATSMuxerClient(natsClient)
	err := client.ListenForResponses()
	if err != nil {
		return nil, err
	}

	return &AuctionNATSClient{
		client:  client,
		timeout: timeout,
		logger:  logger.Session("auction-nats-client"),
	}, nil
}

func (rep *AuctionNATSClient) BidForStartAuction(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	bidLog := rep.logger.Session("start-bid", lager.Data{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
	})

	bidLog.Info("fetching")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).BidForStartAuction)
	}
	payload, _ := json.Marshal(startAuctionInfo)

	responses, _ := rep.aggregateWithTimeout(bidLog, subjects, payload)

	results := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			bidLog.Error("failed-to-unmarshal", err, lager.Data{
				"payload": string(response),
			})
			continue
		}
		results = append(results, bid)
	}

	bidLog.Info("fetched", lager.Data{
		"num-bids-received": len(results),
	})

	return results
}

func (rep *AuctionNATSClient) BidForStopAuction(repGuids []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	bidLog := rep.logger.Session("stop-bid", lager.Data{
		"stop-auction-info": stopAuctionInfo,
		"num-rep-guids":     len(repGuids),
	})

	bidLog.Info("fetching")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).BidForStopAuction)
	}
	payload, _ := json.Marshal(stopAuctionInfo)

	responses, _ := rep.aggregateWithTimeout(bidLog, subjects, payload)

	results := auctiontypes.StopAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StopAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			bidLog.Error("failed-to-unmarshal", err, lager.Data{
				"payload": string(response),
			})
			continue
		}
		results = append(results, bid)
	}

	bidLog.Info("fetched", lager.Data{
		"num-bids-received": len(results),
	})

	return results
}

func (rep *AuctionNATSClient) RebidThenTentativelyReserve(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	bidLog := rep.logger.Session("rebid-then-reserve", lager.Data{
		"start-auction-info": startAuctionInfo,
		"num-rep-guids":      len(repGuids),
	})

	bidLog.Info("fetching")

	subjects := []string{}
	subjectToRepGuid := map[string]string{}
	for _, repGuid := range repGuids {
		subject := nats.NewSubjects(repGuid).RebidThenTentativelyReserve
		subjects = append(subjects, subject)
		subjectToRepGuid[subject] = repGuid
	}
	payload, _ := json.Marshal(startAuctionInfo)

	responses, failedSubjects := rep.aggregateWithTimeout(bidLog, subjects, payload)

	results := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		bid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response, &bid)
		if err != nil {
			bidLog.Error("failed-to-unmarshal", err, lager.Data{
				"payload": string(response),
			})
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

	bidLog.Info("fetched", lager.Data{
		"num-bids-received": len(results),
	})

	return results
}

func (rep *AuctionNATSClient) ReleaseReservation(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	releaseLog := rep.logger.Session("release-reservation", lager.Data{
		"start-auction-info":   startAuctionInfo,
		"rep-guids-to-release": repGuids,
	})

	releaseLog.Info("starting")

	subjects := []string{}
	for _, repGuid := range repGuids {
		subjects = append(subjects, nats.NewSubjects(repGuid).ReleaseReservation)
	}

	payload, _ := json.Marshal(startAuctionInfo)

	rep.aggregateWithTimeout(releaseLog, subjects, payload)

	releaseLog.Info("done")
}

func (rep *AuctionNATSClient) Run(repGuid string, startAuction models.LRPStartAuction) {
	runLog := rep.logger.Session("run", lager.Data{
		"start-auction-info": startAuction,
		"rep-guid":           repGuid,
	})

	runLog.Info("starting")

	subjects := nats.NewSubjects(repGuid)
	payload, _ := json.Marshal(startAuction)
	_, err := rep.publishWithTimeout(subjects.Run, payload)

	if err != nil {
		runLog.Error("failed-to-publish", err)
		return
	}

	runLog.Info("done")
}

func (rep *AuctionNATSClient) Stop(repGuid string, stopInstance models.StopLRPInstance) {
	stopLog := rep.logger.Session("stop", lager.Data{
		"stop-instance": stopInstance,
		"rep-guid":      repGuid,
	})

	stopLog.Info("stopping")

	subjects := nats.NewSubjects(repGuid)
	payload, _ := json.Marshal(stopInstance)

	_, err := rep.publishWithTimeout(subjects.Stop, payload)

	if err != nil {
		stopLog.Error("failed-to-publish", err)
		return
	}

	stopLog.Info("done")
}

func (rep *AuctionNATSClient) publishWithTimeout(subject string, payload []byte) ([]byte, error) {
	response, err := rep.client.Request(subject, payload, rep.timeout)
	if err != nil {
		return nil, err
	}

	if string(response) == "error" {
		return nil, RequestFailedError
	}

	return response, nil
}

func (rep *AuctionNATSClient) aggregateWithTimeout(logger lager.Logger, subjects []string, payload []byte) ([][]byte, []string) {
	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(subjects))

	lock := &sync.Mutex{}
	results := [][]byte{}
	failed := []string{}

	for _, subject := range subjects {
		go func(subject string) {
			defer allReceived.Done()

			result, err := rep.publishWithTimeout(subject, payload)
			if err != nil {
				logger.Error("aggregate-request-publish-failed", err)

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

func (rep *AuctionNATSClient) TotalResources(repGuid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	subjects := nats.NewSubjects(repGuid)
	response, err := rep.publishWithTimeout(subjects.TotalResources, nil)
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

func (rep *AuctionNATSClient) SimulatedInstances(repGuid string) []auctiontypes.SimulatedInstance {
	var instances []auctiontypes.SimulatedInstance
	subjects := nats.NewSubjects(repGuid)
	response, err := rep.publishWithTimeout(subjects.SimulatedInstances, nil)
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

func (rep *AuctionNATSClient) Reset(repGuid string) {
	subjects := nats.NewSubjects(repGuid)
	_, err := rep.publishWithTimeout(subjects.Reset, nil)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}

func (rep *AuctionNATSClient) SetSimulatedInstances(repGuid string, instances []auctiontypes.SimulatedInstance) {
	subjects := nats.NewSubjects(repGuid)
	payload, _ := json.Marshal(instances)
	_, err := rep.publishWithTimeout(subjects.SetSimulatedInstances, payload)
	if err != nil {
		//test only, so panic is OK
		panic(err)
	}
}
