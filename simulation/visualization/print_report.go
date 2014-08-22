package visualization

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
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

func PrintReport(client auctiontypes.SimulationRepPoolClient, results []auctiontypes.StartAuctionResult, representatives []string, duration time.Duration, rules auctiontypes.StartAuctionRules) {
	roundsDistribution := map[int]int{}
	auctionedInstances := map[string]bool{}

	fmt.Printf("Finished %d Auctions among %d Representatives in %s\n", len(results), len(representatives), duration)
	fmt.Println()
	///
	fmt.Println("Rounds Distributions")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		auctionedInstances[result.LRPStartAuction.InstanceGuid] = true
	}

	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %d (%.1f%%)\n", i, roundsDistribution[i], float64(roundsDistribution[i])/float64(len(results))*100.0)
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

	minTime, maxTime, meanTime := time.Hour, time.Duration(0), time.Duration(0)
	for _, result := range results {
		if result.Duration < minTime {
			minTime = result.Duration
		}
		if result.Duration > maxTime {
			maxTime = result.Duration
		}
		meanTime += result.Duration
	}

	meanTime = meanTime / time.Duration(len(results))
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
}
