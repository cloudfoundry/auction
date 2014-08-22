package visualization

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/onsi/gomega/format"
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

func init() {
	format.UseStringerRepresentation = true
}

func PrintReport(client auctiontypes.SimulationRepPoolClient, results []auctiontypes.StartAuctionResult, representatives []string, duration time.Duration, rules auctiontypes.StartAuctionRules) {
	roundsDistribution := map[int]int{}
	roundsBiddingTimeDistributions := map[int][]time.Duration{}
	auctionedInstances := map[string]bool{}

	fmt.Printf("Finished %d Auctions among %d Representatives in %s\n", len(results), len(representatives), duration)
	fmt.Println()
	///
	fmt.Println("Rounds Distributions")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		roundsBiddingTimeDistributions[result.NumRounds] = append(roundsBiddingTimeDistributions[result.NumRounds], result.BiddingDuration)
		auctionedInstances[result.LRPStartAuction.InstanceGuid] = true
	}

	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			minTime, maxTime, meanTime := StatsForDurations(roundsBiddingTimeDistributions[i])
			percentage := fmt.Sprintf("(%.1f%%)", float64(roundsDistribution[i])/float64(len(results))*100.0)
			fmt.Printf("  %3d: %4d %7s Time: min:%16s max:%16s mean:%16s\n", i, roundsDistribution[i], percentage, minTime, maxTime, meanTime)
		}
	}

	///

	fmt.Println("Distribution")
	maxGuidLength := 0
	for _, repGuid := range representatives {
		if len(repGuid) > maxGuidLength {
			maxGuidLength = len(repGuid)
		}
	}
	guidFormat := fmt.Sprintf("%%%ds", maxGuidLength)

	numNew := 0
	for _, repGuid := range representatives {
		repString := fmt.Sprintf(guidFormat, repGuid)

		instanceString := ""
		instances := client.SimulatedInstances(repGuid)

		availableColors := []string{"red", "cyan", "yellow", "gray", "purple", "green"}
		colorLookup := map[string]string{"red": redColor, "green": greenColor, "cyan": cyanColor, "yellow": yellowColor, "gray": lightGrayColor, "purple": purpleColor}

		originalCounts := map[string]int{}
		newCounts := map[string]int{}
		totalUsage := 0
		for _, instance := range instances {
			key := "green"
			if _, ok := colorLookup[instance.ProcessGuid]; ok {
				key = instance.ProcessGuid
			}
			if auctionedInstances[instance.InstanceGuid] {
				newCounts[key] += instance.MemoryMB
				numNew += 1
			} else {
				originalCounts[key] += instance.MemoryMB
			}
			totalUsage += instance.MemoryMB
		}
		for _, col := range availableColors {
			instanceString += strings.Repeat(colorLookup[col]+"-"+defaultStyle, originalCounts[col])
			instanceString += strings.Repeat(colorLookup[col]+"+"+defaultStyle, newCounts[col])
		}
		instanceString += strings.Repeat(grayColor+"."+defaultStyle, client.TotalResources(repGuid).MemoryMB-totalUsage)

		fmt.Printf("  %s: %s\n", repString, instanceString)
	}

	if numNew < len(auctionedInstances) {
		expected := len(auctionedInstances)
		fmt.Printf("%s!!!!MISSING INSTANCES!!!!  Expected %d, got %d (%.3f %% failure rate)%s", redColor, expected, numNew, float64(expected-numNew)/float64(expected), defaultStyle)
	}

	durations := []time.Duration{}
	for _, result := range results {
		durations = append(durations, result.Duration)
	}
	minTime, maxTime, meanTime := StatsForDurations(durations)
	fmt.Printf("%14s  Min: %16s | Max: %16s | Mean: %16s\n", "Wait Times:", minTime, maxTime, meanTime)

	///

	minRounds, maxRounds, totalRounds, meanRounds := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumRounds < minRounds {
			minRounds = result.NumRounds
		}
		if result.NumRounds > maxRounds {
			maxRounds = result.NumRounds
		}
		totalRounds += result.NumRounds
		meanRounds += float64(result.NumRounds)
	}

	meanRounds = meanRounds / float64(len(results))
	fmt.Printf("%14s  Min: %16d | Max: %16d | Mean: %16.2f | Total: %16d\n", "Rounds:", minRounds, maxRounds, meanRounds, totalRounds)

	///

	minCommunications, maxCommunications, totalCommunications, meanCommunications := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumCommunications < minCommunications {
			minCommunications = result.NumCommunications
		}
		if result.NumCommunications > maxCommunications {
			maxCommunications = result.NumCommunications
		}
		totalCommunications += result.NumCommunications
		meanCommunications += float64(result.NumCommunications)
	}

	meanCommunications = meanCommunications / float64(len(results))
	fmt.Printf("%14s  Min: %16d | Max: %16d | Mean: %16.2f | Total: %16d\n", "Communication:", minCommunications, maxCommunications, meanCommunications, totalCommunications)

	firstAuctionTime := results[0].AuctionStartTime
	for _, result := range results {
		if firstAuctionTime.After(result.AuctionStartTime) {
			firstAuctionTime = result.AuctionStartTime
		}
	}

	for _, result := range results {
		if result.BiddingDuration > 20*time.Second {
			fmt.Printf("Starting at %s, %s took %s:\n%s\n", result.AuctionStartTime.Sub(firstAuctionTime), result.LRPStartAuction.InstanceGuid, result.Duration, format.Object(result.Events, 1))
		}
	}
}

func StatsForDurations(durations []time.Duration) (time.Duration, time.Duration, time.Duration) {
	minTime, maxTime, meanTime := time.Hour, time.Duration(0), time.Duration(0)
	for _, duration := range durations {
		if duration < minTime {
			minTime = duration
		}
		if duration > maxTime {
			maxTime = duration
		}
		meanTime += duration
	}
	if len(durations) > 0 {
		meanTime = meanTime / time.Duration(len(durations))
	}

	return minTime, maxTime, meanTime
}
