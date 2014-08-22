package visualization

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/GaryBoone/GoStats/stats"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/communication/inprocess"
)

const defaultStyle = "\x1b[0m"
const boldStyle = "\x1b[1m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"
const grayColor = "\x1b[90m"
const lightGrayColor = "\x1b[37m"
const purpleColor = "\x1b[35m"

func PrintReport(
	client auctiontypes.SimulationRepPoolClient,
	results []auctiontypes.StartAuctionResult,
	representatives []string,
	duration time.Duration,
	rules auctiontypes.StartAuctionRules,
) {
	fmt.Printf("\nFinished %d Auctions among %d Representatives in %s\n", len(results), len(representatives), duration)
	fmt.Printf("  %#v\n", rules)
	if _, ok := client.(*inprocess.InprocessClient); ok {
		fmt.Printf("  Latency Range: %s < %s, Timeout: %s\n", inprocess.LatencyMin, inprocess.LatencyMax, inprocess.Timeout)
	}

	fmt.Println("\n*** AUCTION STATISTICS ***")
	roundsData := []float64{}
	roundsDistribution := map[int]int{}
	communicationsData := []float64{}
	biddingTimesData := []float64{}
	waitTimesData := []float64{}

	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		roundsData = append(roundsData, float64(result.NumRounds))
		communicationsData = append(communicationsData, float64(result.NumCommunications))
		biddingTimesData = append(biddingTimesData, float64(result.BiddingDuration.Seconds()*1000.0))
		waitTimesData = append(waitTimesData, float64(result.Duration.Seconds()*1000.0))
	}

	fmt.Println("\nNumber of Rounds")
	roundsStats := stats.Stats{}
	roundsStats.UpdateArray(roundsData)
	fmt.Printf("  Min: %.0f | Max: %.0f | Total: %.0f | Mean: %.2f | Variance: %.2f\n", roundsStats.Min(), roundsStats.Max(), roundsStats.Sum(), roundsStats.Mean(), roundsStats.PopulationVariance())
	fmt.Println("  Distribution:")
	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %s\n", i, strings.Repeat("■", roundsDistribution[i]))
		}
	}

	fmt.Println("\nNumber of Communications")
	communicationsStats := stats.Stats{}
	communicationsStats.UpdateArray(communicationsData)
	fmt.Printf("  Min: %.0f | Max: %.0f | Total: %.0f | Mean: %.2f | Variance: %.2f\n", communicationsStats.Min(), communicationsStats.Max(), communicationsStats.Sum(), communicationsStats.Mean(), communicationsStats.PopulationVariance())

	fmt.Println("\nBidding Times")
	biddingTimesStats := stats.Stats{}
	biddingTimesStats.UpdateArray(biddingTimesData)
	fmt.Printf("  Min: %.3fms | Max: %.3fms | Total: %.3fms | Mean: %.3fms | Variance: %.3fms\n", biddingTimesStats.Min(), biddingTimesStats.Max(), biddingTimesStats.Sum(), biddingTimesStats.Mean(), biddingTimesStats.PopulationVariance())

	fmt.Println("\nWait Times")
	waitTimesStats := stats.Stats{}
	waitTimesStats.UpdateArray(waitTimesData)
	fmt.Printf("  Min: %.3fms | Max: %.3fms | Total: %.3fms | Mean: %.3fms | Variance: %.3fms\n", waitTimesStats.Min(), waitTimesStats.Max(), waitTimesStats.Sum(), waitTimesStats.Mean(), waitTimesStats.PopulationVariance())

	fmt.Println("\n*** APP STATISTICS ***")
	excessMaxColocationFactorData := []float64{}

	maxNumColocatedInstancesForProcess := make(map[string]int)
	totalNumInstancesForProcess := make(map[string]int)
	for _, repGuid := range representatives {
		numColocatedInstancesForProcess := make(map[string]int)

		for _, instance := range client.SimulatedInstances(repGuid) {
			numColocatedInstancesForProcess[instance.ProcessGuid] += 1
		}

		for processGuid, numColocatedInstances := range numColocatedInstancesForProcess {
			if numColocatedInstances > maxNumColocatedInstancesForProcess[processGuid] {
				maxNumColocatedInstancesForProcess[processGuid] = numColocatedInstances
			}
			totalNumInstancesForProcess[processGuid] += numColocatedInstances
		}
	}
	for processGuid, maxNumColocatedInstances := range maxNumColocatedInstancesForProcess {
		expectedMaxNumColocatedInstances := math.Ceil(float64(totalNumInstancesForProcess[processGuid]) / float64(len(representatives)))
		excessMaxColocationFactorData = append(
			excessMaxColocationFactorData,
			float64(maxNumColocatedInstances)/expectedMaxNumColocatedInstances,
		)
	}

	fmt.Println("\nExcess Colocation Factors")
	excessMaxColocationFactorStats := stats.Stats{}
	excessMaxColocationFactorStats.UpdateArray(excessMaxColocationFactorData)
	fmt.Printf("  Min: %.4f | Max: %.4f | Mean: %.4f | Variance: %.4f\n", excessMaxColocationFactorStats.Min(), excessMaxColocationFactorStats.Max(), excessMaxColocationFactorStats.Mean(), excessMaxColocationFactorStats.PopulationVariance())

	fmt.Println("\n*** REP STATISTICS ***")
	memoryData := []float64{}
	diskData := []float64{}
	containersData := []float64{}
	maxGuidLength := 0
	for _, repGuid := range representatives {
		if len(repGuid) > maxGuidLength {
			maxGuidLength = len(repGuid)
		}
	}
	guidFormat := fmt.Sprintf("%%%ds", maxGuidLength)
	containerHistogramLines := []string{}
	auctionedInstances := map[string]bool{}
	for _, result := range results {
		auctionedInstances[result.LRPStartAuction.InstanceGuid] = true
	}
	numNew := 0

	for _, repGuid := range representatives {
		instances := client.SimulatedInstances(repGuid)

		memory, disk := 0, 0
		containersData = append(containersData, float64(len(instances)))

		availableColors := []string{"red", "cyan", "yellow", "gray", "purple", "green"}
		colorLookup := map[string]string{"red": redColor, "green": greenColor, "cyan": cyanColor, "yellow": yellowColor, "gray": lightGrayColor, "purple": purpleColor}
		originalCounts := map[string]int{}
		newCounts := map[string]int{}

		for _, instance := range instances {
			memory += instance.MemoryMB
			disk += instance.DiskMB

			key := "green"
			if _, ok := colorLookup[instance.ProcessGuid]; ok {
				key = instance.ProcessGuid
			}
			if auctionedInstances[instance.InstanceGuid] {
				newCounts[key] += 1
				numNew += 1
			} else {
				originalCounts[key] += 1
			}
		}

		memoryData = append(memoryData, float64(memory))
		diskData = append(diskData, float64(disk))

		instanceString := ""
		for _, col := range availableColors {
			instanceString += strings.Repeat(colorLookup[col]+"○"+defaultStyle, originalCounts[col])
			instanceString += strings.Repeat(colorLookup[col]+"●"+defaultStyle, newCounts[col])
		}
		instanceString += strings.Repeat(grayColor+"○"+defaultStyle, client.TotalResources(repGuid).Containers-len(instances))

		containerHistogramLines = append(containerHistogramLines, fmt.Sprintf("  %s: %s", fmt.Sprintf(guidFormat, repGuid), instanceString))
	}

	fmt.Println("\nUsed Memory")
	memoryStats := stats.Stats{}
	memoryStats.UpdateArray(memoryData)
	fmt.Printf("  Min: %.0f | Max: %.0f | Total: %.0f | Mean: %.2f | Variance: %.2f\n", memoryStats.Min(), memoryStats.Max(), memoryStats.Sum(), memoryStats.Mean(), memoryStats.PopulationVariance())

	fmt.Println("\nUsed Disk")
	diskStats := stats.Stats{}
	diskStats.UpdateArray(diskData)
	fmt.Printf("  Min: %.0f | Max: %.0f | Total: %.0f | Mean: %.2f | Variance: %.2f\n", diskStats.Min(), diskStats.Max(), diskStats.Sum(), diskStats.Mean(), diskStats.PopulationVariance())

	fmt.Println("\nNumber of Running Containers")
	containersStats := stats.Stats{}
	containersStats.UpdateArray(containersData)
	fmt.Printf("  Min: %.0f | Max: %.0f | Total: %.0f | Mean: %.2f | Variance: %.2f\n", containersStats.Min(), containersStats.Max(), containersStats.Sum(), containersStats.Mean(), containersStats.PopulationVariance())
	fmt.Println("  Distribution:")
	for _, histogramLine := range containerHistogramLines {
		fmt.Println(histogramLine)
	}
	if numNew < len(auctionedInstances) {
		expected := len(auctionedInstances)
		fmt.Printf("  %s!!!!MISSING INSTANCES!!!!  Expected %d, got %d (%.3f %% failure rate)%s", redColor, expected, numNew, float64(expected-numNew)/float64(expected), defaultStyle)
	}
}
