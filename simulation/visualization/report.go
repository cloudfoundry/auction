package visualization

import (
	"sort"
	"sync"
	"time"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/workpool"
)

type Report struct {
	Cells                        map[string]auctiontypes.AuctionRep
	NumAuctions                  int
	AuctionResults               auctionrunner.WorkResults
	AuctionDuration              time.Duration
	CellStates                   map[string]auctiontypes.RepState
	InstancesByRep               map[string][]auctiontypes.LRP
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

func NewReport(numAuctions int, cells map[string]auctiontypes.AuctionRep, results auctionrunner.WorkResults, duration time.Duration) *Report {
	states := fetchStates(cells)
	return &Report{
		Cells:           cells,
		NumAuctions:     numAuctions,
		AuctionResults:  results,
		AuctionDuration: duration,
		CellStates:      states,
		InstancesByRep:  instancesByRepFromStates(states),
	}
}

func (r *Report) IsAuctionedInstance(inst auctiontypes.LRP) bool {
	if r.auctionedInstancesByInstGuid == nil {
		r.auctionedInstancesByInstGuid = map[string]bool{}
		for _, result := range r.AuctionResults.SuccessfulStarts {
			r.auctionedInstancesByInstGuid[result.LRPStartAuction.InstanceGuid] = true
		}
	}

	return r.auctionedInstancesByInstGuid[inst.InstanceGuid]
}

func (r *Report) AuctionsPerformed() int {
	return len(r.AuctionResults.SuccessfulStarts) + len(r.AuctionResults.FailedStarts)
}

func (r *Report) NReps() int {
	return len(r.Cells)
}

func (r *Report) NMissingInstances() int {
	return len(r.AuctionResults.FailedStarts)
}

func (r *Report) InitialDistributionScore() float64 {
	memoryCounts := []float64{}
	for _, instances := range r.InstancesByRep {
		memoryCount := 0.0
		for _, instance := range instances {
			if !r.IsAuctionedInstance(instance) {
				memoryCount += float64(instance.MemoryMB)
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
			memoryCount += float64(instance.MemoryMB)
		}
		memoryCounts = append(memoryCounts, memoryCount)
	}

	return stats.StatsPopulationStandardDeviation(memoryCounts) / stats.StatsMean(memoryCounts)
}

func (r *Report) AuctionsPerSecond() float64 {
	return float64(r.AuctionsPerformed()) / r.AuctionDuration.Seconds()
}

func (r *Report) WaitTimeStats() Stat {
	waitTimes := []float64{}
	for _, result := range r.AuctionResults.SuccessfulStarts {
		waitTimes = append(waitTimes, result.WaitDuration.Seconds())
	}

	return NewStat(waitTimes)
}

func fetchStates(cells map[string]auctiontypes.AuctionRep) map[string]auctiontypes.RepState {
	workPool := workpool.NewWorkPool(50)
	wg := &sync.WaitGroup{}
	wg.Add(len(cells))
	lock := &sync.Mutex{}
	states := map[string]auctiontypes.RepState{}
	for repGuid, cell := range cells {
		repGuid := repGuid
		cell := cell
		workPool.Submit(func() {
			state, _ := cell.State()
			lock.Lock()
			states[repGuid] = state
			lock.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
	workPool.Stop()
	return states
}

func instancesByRepFromStates(states map[string]auctiontypes.RepState) map[string][]auctiontypes.LRP {
	instancesByRepGuid := map[string][]auctiontypes.LRP{}
	for repGuid, state := range states {
		instances := state.LRPs
		sort.Sort(ByProcessGuid(instances))
		instancesByRepGuid[repGuid] = instances
	}

	return instancesByRepGuid
}

type ByProcessGuid []auctiontypes.LRP

func (a ByProcessGuid) Len() int           { return len(a) }
func (a ByProcessGuid) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProcessGuid) Less(i, j int) bool { return a[i].ProcessGuid < a[j].ProcessGuid }
