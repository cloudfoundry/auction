package visualization

import (
	"sort"
	"time"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/onsi/auction/types"
)

type Report struct {
	RepGuids                     []string
	AuctionResults               []types.AuctionResult
	InstancesByRep               map[string][]types.Instance
	AuctionDuration              time.Duration
	auctionedInstancesByInstGuid map[string]bool
}

type Stat struct {
	Min    float64
	Max    float64
	Mean   float64
	StdDev float64
	Total  float64
}

func NewStat(data []float64) Stat {
	return Stat{
		Min:    stats.StatsMin(data),
		Max:    stats.StatsMax(data),
		Mean:   stats.StatsMean(data),
		StdDev: stats.StatsPopulationStandardDeviation(data),
		Total:  stats.StatsSum(data),
	}
}

func (r *Report) IsAuctionedInstance(inst types.Instance) bool {
	if r.auctionedInstancesByInstGuid == nil {
		r.auctionedInstancesByInstGuid = map[string]bool{}
		for _, result := range r.AuctionResults {
			r.auctionedInstancesByInstGuid[result.Instance.InstanceGuid] = true
		}
	}

	return r.auctionedInstancesByInstGuid[inst.InstanceGuid]
}

func (r *Report) NAuctions() int {
	return len(r.AuctionResults)
}

func (r *Report) NReps() int {
	return len(r.RepGuids)
}

func (r *Report) NMissingInstances() int {
	numRunningThatWereAuctioned := 0
	for _, instances := range r.InstancesByRep {
		for _, instance := range instances {
			if r.IsAuctionedInstance(instance) {
				numRunningThatWereAuctioned += 1
			}
		}
	}

	return len(r.AuctionResults) - numRunningThatWereAuctioned
}

func (r *Report) InitialDistributionScore() float64 {
	memoryCounts := []float64{}
	for _, instances := range r.InstancesByRep {
		memoryCount := 0.0
		for _, instance := range instances {
			if !r.IsAuctionedInstance(instance) {
				memoryCount += instance.Resources.MemoryMB
			}
		}
		memoryCounts = append(memoryCounts, memoryCount)
	}

	if stats.StatsSum(memoryCounts) == 0 {
		return 0
	}

	return stats.StatsPopulationStandardDeviation(memoryCounts) / stats.StatsMean(memoryCounts)
}

func (r *Report) DistributionScore() float64 {
	memoryCounts := []float64{}
	for _, instances := range r.InstancesByRep {
		memoryCount := 0.0
		for _, instance := range instances {
			memoryCount += instance.Resources.MemoryMB
		}
		memoryCounts = append(memoryCounts, memoryCount)
	}

	return stats.StatsPopulationStandardDeviation(memoryCounts) / stats.StatsMean(memoryCounts)
}

func (r *Report) AuctionsPerSecond() float64 {
	return float64(r.NAuctions()) / r.AuctionDuration.Seconds()
}

func (r *Report) CommStats() Stat {
	comms := []float64{}
	for _, result := range r.AuctionResults {
		comms = append(comms, float64(result.NumCommunications))
	}

	return NewStat(comms)
}

func (r *Report) BiddingTimeStats() Stat {
	biddingTimes := []float64{}
	for _, result := range r.AuctionResults {
		biddingTimes = append(biddingTimes, result.BiddingDuration.Seconds())
	}

	return NewStat(biddingTimes)
}

func (r *Report) WaitTimeStats() Stat {
	waitTimes := []float64{}
	for _, result := range r.AuctionResults {
		waitTimes = append(waitTimes, result.Duration.Seconds())
	}

	return NewStat(waitTimes)
}

func FetchAndSortInstances(client types.TestRepPoolClient, repGuids []string) map[string][]types.Instance {
	instancesByRepGuid := map[string][]types.Instance{}
	for _, guid := range repGuids {
		instances := client.Instances(guid)
		sort.Sort(ByAppGuid(instances))
		instancesByRepGuid[guid] = instances
	}

	return instancesByRepGuid
}

type ByAppGuid []types.Instance

func (a ByAppGuid) Len() int           { return len(a) }
func (a ByAppGuid) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAppGuid) Less(i, j int) bool { return a[i].AppGuid < a[j].AppGuid }
