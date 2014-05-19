package inprocess

import (
	"time"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
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

func (client *InprocessClient) TotalResources(guid string) types.Resources {
	return client.reps[guid].TotalResources()
}

func (client *InprocessClient) Instances(guid string) []types.Instance {
	return client.reps[guid].Instances()
}

func (client *InprocessClient) SetInstances(guid string, instances []types.Instance) {
	client.reps[guid].SetInstances(instances)
}

func (client *InprocessClient) Reset(guid string) {
	client.reps[guid].Reset()
}

func (client *InprocessClient) score(guid string, instance types.Instance, c chan types.ScoreResult) {
	result := types.ScoreResult{
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

func (client *InprocessClient) Score(representatives []string, instance types.Instance) types.ScoreResults {
	c := make(chan types.ScoreResult)
	for _, guid := range representatives {
		go client.score(guid, instance, c)
	}

	results := types.ScoreResults{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastScore(guid string, instance types.Instance, c chan types.ScoreResult) {
	result := types.ScoreResult{
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

func (client *InprocessClient) ScoreThenTentativelyReserve(guids []string, instance types.Instance) types.ScoreResults {
	c := make(chan types.ScoreResult)
	for _, guid := range guids {
		go client.reserveAndRecastScore(guid, instance, c)
	}

	results := types.ScoreResults{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) ReleaseReservation(guids []string, instance types.Instance) {
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

func (client *InprocessClient) Claim(guid string, instance types.Instance) {
	client.beSlowAndPossiblyTimeout(guid)

	client.reps[guid].Claim(instance)
}
