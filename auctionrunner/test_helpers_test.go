package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/rep"
	. "github.com/onsi/gomega"
)

func BuildLRPStartRequest(processGuid, domain string, indices []int, rootFS string, memoryMB, diskMB int32) auctioneer.LRPStartRequest {
	return auctioneer.NewLRPStartRequest(processGuid, domain, indices, rep.NewResource(memoryMB, diskMB, rootFS))
}

func BuildTaskStartRequest(taskGuid, domain, rootFS string, memoryMB, diskMB int32) auctioneer.TaskStartRequest {
	return auctioneer.NewTaskStartRequest(*BuildTask(taskGuid, domain, rootFS, memoryMB, diskMB))
}

func BuildLRP(guid, domain string, index int, rootFS string, memoryMB, diskMB int32) *rep.LRP {
	lrpKey := models.NewActualLRPKey(guid, int32(index), domain)
	lrp := rep.NewLRP(lrpKey, rep.NewResource(memoryMB, diskMB, rootFS))
	return &lrp
}

func BuildTask(taskGuid, domain, rootFS string, memoryMB, diskMB int32) *rep.Task {
	task := rep.NewTask(taskGuid, domain, rep.NewResource(memoryMB, diskMB, rootFS))
	return &task
}

func BuildLRPAuction(processGuid, domain string, index int, rootFS string, memoryMB, diskMB int32, queueTime time.Time) auctiontypes.LRPAuction {
	lrpKey := models.NewActualLRPKey(processGuid, int32(index), domain)
	return auctiontypes.NewLRPAuction(rep.NewLRP(lrpKey, rep.NewResource(memoryMB, diskMB, rootFS)), queueTime)
}

func BuildLRPAuctionWithPlacementError(processGuid, domain string, index int, rootFS string, memoryMB, diskMB int32, queueTime time.Time, placementError string) auctiontypes.LRPAuction {
	lrpKey := models.NewActualLRPKey(processGuid, int32(index), domain)
	a := auctiontypes.NewLRPAuction(rep.NewLRP(lrpKey, rep.NewResource(memoryMB, diskMB, rootFS)), queueTime)
	a.PlacementError = placementError
	return a
}

func BuildLRPAuctions(start auctioneer.LRPStartRequest, queueTime time.Time) []auctiontypes.LRPAuction {
	auctions := make([]auctiontypes.LRPAuction, 0, len(start.Indices))
	for _, index := range start.Indices {
		lrpKey := models.NewActualLRPKey(start.ProcessGuid, int32(index), start.Domain)
		auctions = append(auctions, auctiontypes.NewLRPAuction(rep.NewLRP(lrpKey, start.Resource), queueTime))
	}

	return auctions
}

func BuildTaskAuction(task *rep.Task, queueTime time.Time) auctiontypes.TaskAuction {
	return auctiontypes.NewTaskAuction(*task, queueTime)
}

const linuxStack = "linux"

var linuxRootFSURL = models.PreloadedRootFS(linuxStack)

var linuxOnlyRootFSProviders = rep.RootFSProviders{models.PreloadedRootFSScheme: rep.NewFixedSetRootFSProvider(linuxStack)}

const windowsStack = "windows"

var windowsRootFSURL = models.PreloadedRootFS(windowsStack)

var windowsOnlyRootFSProviders = rep.RootFSProviders{models.PreloadedRootFSScheme: rep.NewFixedSetRootFSProvider(windowsStack)}

func BuildCellState(
	zone string,
	memoryMB int32,
	diskMB int32,
	containers int,
	evacuating bool,
	startingContainerCount int,
	rootFSProviders rep.RootFSProviders,
	lrps []rep.LRP,
) rep.CellState {
	totalResources := rep.NewResources(memoryMB, diskMB, containers)

	availableResources := totalResources.Copy()
	for i := range lrps {
		availableResources.Subtract(&lrps[i].Resource)
	}

	Expect(availableResources.MemoryMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.DiskMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.Containers).To(BeNumerically(">=", 0), "Check your math!")

	return rep.NewCellState(rootFSProviders, availableResources, totalResources, lrps, nil, zone, startingContainerCount, evacuating, []string{})
}
