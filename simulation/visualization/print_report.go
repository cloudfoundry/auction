package visualization

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/workpool"
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

func cellID(index) string {
	return fmt.Sprintf("REP-%d", index+1)
}

func PrintReport(report *Report) {
	if report.AuctionsPerformed() == 0 {
		fmt.Println("Got no results!")
		return
	}

	fmt.Printf("Finished %d Auctions (%d succeeded, %d failed) among %d Cells in %s\n", report.AuctionsPerformed(), len(r.AuctionResults.SuccessfulStarts), len(r.AuctionResults.FailedStarts), len(report.Cells), report.AuctionDuration)
	fmt.Println()

	auctionedInstances := map[string]bool{}
	for _, start := range report.AuctionResults.SuccessfulStarts {
		auctionedInstances[start.LRPStartAuction.InstanceGuid] = true
	}

	fmt.Println("Distribution")
	maxGuidLength := cellID(len(report.Cells) - 1)
	guidFormat := fmt.Sprintf("%%%ds", maxGuidLength)

	numNew := 0
	for _, repAddress := range representatives {
		repString := fmt.Sprintf(guidFormat, repAddress.RepGuid)

		instanceString := ""
		instances := reportData[repAddress.RepGuid].Instances

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
		instanceString += strings.Repeat(grayColor+"."+defaultStyle, reportData[repAddress.RepGuid].TotalResources.MemoryMB-totalUsage)

		fmt.Printf("  %s: %s\n", repString, instanceString)
	}

	if numNew < len(auctionedInstances) {
		fmt.Printf("%s!!!!MISSING INSTANCES!!!!  Expected %d, got %d (%.3f %% failure rate)%s", redColor, expectedAuctionCount, numNew, float64(expectedAuctionCount-numNew)/float64(expectedAuctionCount), defaultStyle)
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
		if result.BiddingDuration > time.Hour { //turn this on to get detailed logs about auctions that took a long time.
			fmt.Printf("Starting at %s, %s took %s:\n%s\n", result.AuctionStartTime.Sub(firstAuctionTime), result.LRPStartAuction.InstanceGuid, result.Duration, format.Object(result.Events, 1))
		}
	}
}

func prefetchReportData(client auctiontypes.SimulationRepPoolClient, representatives []auctiontypes.RepAddress) map[string]ReportData {
	workPool := workpool.NewWorkPool(50)
	wg := &sync.WaitGroup{}
	wg.Add(len(representatives))
	reportDataLock := &sync.Mutex{}
	reportData := map[string]ReportData{}
	for _, repAddress := range representatives {
		repAddress := repAddress
		workPool.Submit(func() {
			instances := client.SimulatedInstances(repAddress)
			resources := client.TotalResources(repAddress)
			reportDataLock.Lock()
			reportData[repAddress.RepGuid] = ReportData{
				Instances:      instances,
				TotalResources: resources,
			}
			reportDataLock.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
	workPool.Stop()
	return reportData
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
