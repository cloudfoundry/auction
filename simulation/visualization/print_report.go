package visualization

import (
	"fmt"
	"strings"
	"time"

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

func PrintReport(client auctiontypes.TestRepPoolClient, results []auctiontypes.AuctionResult, representatives []string, duration time.Duration, rules auctiontypes.AuctionRules) {
	roundsDistribution := map[int]int{}
	auctionedInstances := map[string]bool{}

	///
	fmt.Println("Rounds Distributions")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		auctionedInstances[result.Instance.InstanceGuid] = true
	}

	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %s\n", i, strings.Repeat("■", roundsDistribution[i]))
		}
	}

	///

	fmt.Println("Distribution")
	maxGuidLength := 0
	for _, guid := range representatives {
		if len(guid) > maxGuidLength {
			maxGuidLength = len(guid)
		}
	}
	guidFormat := fmt.Sprintf("%%%ds", maxGuidLength)

	numNew := 0
	for _, guid := range representatives {
		repString := fmt.Sprintf(guidFormat, guid)

		instanceString := ""
		instances := client.Instances(guid)

		availableColors := []string{"red", "cyan", "yellow", "gray", "purple", "green"}
		colorLookup := map[string]string{"red": redColor, "green": greenColor, "cyan": cyanColor, "yellow": yellowColor, "gray": lightGrayColor, "purple": purpleColor}

		originalCounts := map[string]int{}
		newCounts := map[string]int{}
		for _, instance := range instances {
			key := "green"
			if _, ok := colorLookup[instance.AppGuid]; ok {
				key = instance.AppGuid
			}
			if auctionedInstances[instance.InstanceGuid] {
				newCounts[key] += 1
				numNew += 1
			} else {
				originalCounts[key] += 1
			}
		}
		for _, col := range availableColors {
			instanceString += strings.Repeat(colorLookup[col]+"○"+defaultStyle, originalCounts[col])
			instanceString += strings.Repeat(colorLookup[col]+"●"+defaultStyle, newCounts[col])
		}
		instanceString += strings.Repeat(grayColor+"○"+defaultStyle, client.TotalResources(guid).Containers-len(instances))

		fmt.Printf("  %s: %s\n", repString, instanceString)
	}

	fmt.Printf("Finished %d Auctions among %d Representatives in %s\n", len(results), len(representatives), duration)
	if numNew < len(auctionedInstances) {
		expected := len(auctionedInstances)
		fmt.Printf("  %s!!!!MISSING INSTANCES!!!!  Expected %d, got %d (%.3f %% failure rate)%s", redColor, expected, numNew, float64(expected-numNew)/float64(expected), defaultStyle)
	}
	fmt.Printf("  %#v\n", rules)
	if _, ok := client.(*inprocess.InprocessClient); ok {
		fmt.Printf("  Latency Range: %s < %s, Timeout: %s, Flakiness: %.2f\n", inprocess.LatencyMin, inprocess.LatencyMax, inprocess.Timeout)
	}

	///

	fmt.Println("Bidding Times")
	minBiddingTime, maxBiddingTime, totalBiddingTime, meanBiddingTime := time.Hour, time.Duration(0), time.Duration(0), time.Duration(0)
	for _, result := range results {
		if result.BiddingDuration < minBiddingTime {
			minBiddingTime = result.BiddingDuration
		}
		if result.BiddingDuration > maxBiddingTime {
			maxBiddingTime = result.BiddingDuration
		}
		totalBiddingTime += result.BiddingDuration
		meanBiddingTime += result.BiddingDuration
	}

	meanBiddingTime = meanBiddingTime / time.Duration(len(results))
	fmt.Printf("  Min: %s | Max: %s | Total: %s | Mean: %s\n", minBiddingTime, maxBiddingTime, totalBiddingTime, meanBiddingTime)

	fmt.Println("Wait Times")
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
	fmt.Printf("  Min: %s | Max: %s | Mean: %s\n", minTime, maxTime, meanTime)

	///

	fmt.Println("Rounds")
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
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minRounds, maxRounds, totalRounds, meanRounds)

	///

	fmt.Println("Scores")
	minScores, maxScores, totalScores, meanScores := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumCommunications < minScores {
			minScores = result.NumCommunications
		}
		if result.NumCommunications > maxScores {
			maxScores = result.NumCommunications
		}
		totalScores += result.NumCommunications
		meanScores += float64(result.NumCommunications)
	}

	meanScores = meanScores / float64(len(results))
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minScores, maxScores, totalScores, meanScores)

}
