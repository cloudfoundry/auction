package inprocess

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration

type InprocessClient struct {
	reps map[string]*auctionrep.AuctionRep
}

func New(reps map[string]*auctionrep.AuctionRep) *InprocessClient {
	return &InprocessClient{
		reps: reps,
	}
}

func randomSleep(min time.Duration, max time.Duration, timeout time.Duration) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(max-min) + float64(min))
	if sleepDuration <= timeout {
		time.Sleep(sleepDuration)
		return true
	} else {
		time.Sleep(timeout)
		return false
	}
}

func (client *InprocessClient) beSlowAndPossiblyTimeout(guid string) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(LatencyMax-LatencyMin) + float64(LatencyMin))

	if sleepDuration <= Timeout {
		time.Sleep(sleepDuration)
		return false
	} else {
		time.Sleep(Timeout)
		return true
	}
}

func (client *InprocessClient) TotalResources(guid string) auctiontypes.Resources {
	return client.reps[guid].TotalResources()
}

func (client *InprocessClient) SimulatedInstances(guid string) []auctiontypes.SimulatedInstance {
	return client.reps[guid].SimulatedInstances()
}

func (client *InprocessClient) SetSimulatedInstances(guid string, instances []auctiontypes.SimulatedInstance) {
	client.reps[guid].SetSimulatedInstances(instances)
}

func (client *InprocessClient) Reset(guid string) {
	client.reps[guid].Reset()
}

func (client *InprocessClient) startAuctionBid(guid string, startAuctionInfo auctiontypes.StartAuctionInfo, c chan auctiontypes.StartAuctionBid) {
	result := auctiontypes.StartAuctionBid{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timeout"
		return
	}

	bid, err := client.reps[guid].BidForStartAuction(startAuctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Bid = bid
	return
}

func (client *InprocessClient) stopScore(guid string, auctionInfo auctiontypes.StopAuctionInfo, c chan auctiontypes.StopAuctionBid) {
	result := auctiontypes.StopAuctionBid{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timeout"
		return
	}

	bid, instanceGuids, err := client.reps[guid].BidForStopAuction(auctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.InstanceGuids = instanceGuids
	result.Bid = bid
}

func (client *InprocessClient) BidForStopAuction(representatives []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	c := make(chan auctiontypes.StopAuctionBid)
	for _, guid := range representatives {
		go client.stopScore(guid, stopAuctionInfo, c)
	}

	results := auctiontypes.StopAuctionBids{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) BidForStartAuction(representatives []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, guid := range representatives {
		go client.startAuctionBid(guid, startAuctionInfo, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastScore(guid string, startAuctionInfo auctiontypes.StartAuctionInfo, c chan auctiontypes.StartAuctionBid) {
	result := auctiontypes.StartAuctionBid{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timedout"
		return
	}

	bid, err := client.reps[guid].RebidThenTentativelyReserve(startAuctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Bid = bid
	return
}

func (client *InprocessClient) RebidThenTentativelyReserve(guids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, guid := range guids {
		go client.reserveAndRecastScore(guid, startAuctionInfo, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) ReleaseReservation(guids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	c := make(chan bool)
	for _, guid := range guids {
		go func(guid string) {
			client.beSlowAndPossiblyTimeout(guid)
			client.reps[guid].ReleaseReservation(startAuctionInfo)
			c <- true
		}(guid)
	}

	for _ = range guids {
		<-c
	}
}

func (client *InprocessClient) Run(guid string, startAuctionInfo models.LRPStartAuction) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Run(startAuctionInfo)
}

func (client *InprocessClient) Stop(guid string, instanceGuid string) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Stop(instanceGuid)
}
