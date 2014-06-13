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

func (client *InprocessClient) score(guid string, instance auctiontypes.LRPAuctionInfo, c chan auctiontypes.ScoreResult) {
	result := auctiontypes.ScoreResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timeout"
		return
	}

	score, err := client.reps[guid].Score(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (client *InprocessClient) stopScore(guid string, auctionInfo auctiontypes.LRPStopAuctionInfo, c chan auctiontypes.StopScoreResult) {
	result := auctiontypes.StopScoreResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timeout"
		return
	}

	score, instanceGuids, err := client.reps[guid].StopScore(auctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.InstanceGuids = instanceGuids
	result.Score = score
}

func (client *InprocessClient) StopScore(representatives []string, stopAuctionInfo auctiontypes.LRPStopAuctionInfo) auctiontypes.StopScoreResults {
	c := make(chan auctiontypes.StopScoreResult)
	for _, guid := range representatives {
		go client.stopScore(guid, stopAuctionInfo, c)
	}

	results := auctiontypes.StopScoreResults{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) Score(representatives []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
	c := make(chan auctiontypes.ScoreResult)
	for _, guid := range representatives {
		go client.score(guid, instance, c)
	}

	results := auctiontypes.ScoreResults{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastScore(guid string, instance auctiontypes.LRPAuctionInfo, c chan auctiontypes.ScoreResult) {
	result := auctiontypes.ScoreResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	if client.beSlowAndPossiblyTimeout(guid) {
		result.Error = "timedout"
		return
	}

	score, err := client.reps[guid].ScoreThenTentativelyReserve(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Score = score
	return
}

func (client *InprocessClient) ScoreThenTentativelyReserve(guids []string, instance auctiontypes.LRPAuctionInfo) auctiontypes.ScoreResults {
	c := make(chan auctiontypes.ScoreResult)
	for _, guid := range guids {
		go client.reserveAndRecastScore(guid, instance, c)
	}

	results := auctiontypes.ScoreResults{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) ReleaseReservation(guids []string, instance auctiontypes.LRPAuctionInfo) {
	c := make(chan bool)
	for _, guid := range guids {
		go func(guid string) {
			client.beSlowAndPossiblyTimeout(guid)
			client.reps[guid].ReleaseReservation(instance)
			c <- true
		}(guid)
	}

	for _ = range guids {
		<-c
	}
}

func (client *InprocessClient) Run(guid string, instance models.LRPStartAuction) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Run(instance)
}

func (client *InprocessClient) Stop(guid string, instanceGuid string) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Stop(instanceGuid)
}
