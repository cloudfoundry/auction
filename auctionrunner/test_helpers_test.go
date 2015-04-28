package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/gomega"
)

func BuildLRPStartRequest(processGuid string, indices []uint, rootFS string, memoryMB, diskMB int) models.LRPStartRequest {
	return models.LRPStartRequest{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			RootFS:      rootFS,
		},
		Indices: indices,
	}
}

func BuildTask(taskGuid, rootFS string, memoryMB, diskMB int) models.Task {
	return models.Task{
		TaskGuid: taskGuid,
		RootFS:   rootFS,
		MemoryMB: memoryMB,
		DiskMB:   diskMB,
	}
}

func BuildLRPAuction(processGuid string, index int, rootFS string, memoryMB, diskMB int, queueTime time.Time) auctiontypes.LRPAuction {
	return auctiontypes.LRPAuction{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			RootFS:      rootFS,
		},
		Index: index,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: queueTime,
		},
	}
}

func BuildLRPAuctionWithPlacementError(processGuid string, index int, rootFS string, memoryMB, diskMB int, queueTime time.Time, placementError string) auctiontypes.LRPAuction {
	return auctiontypes.LRPAuction{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			RootFS:      rootFS,
		},
		Index: index,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime:      queueTime,
			PlacementError: placementError,
		},
	}
}

func BuildLRPAuctions(lrpStart models.LRPStartRequest, queueTime time.Time) []auctiontypes.LRPAuction {
	auctions := make([]auctiontypes.LRPAuction, 0, len(lrpStart.Indices))
	for _, i := range lrpStart.Indices {
		auctions = append(auctions, auctiontypes.LRPAuction{
			DesiredLRP: lrpStart.DesiredLRP,
			Index:      int(i),
			AuctionRecord: auctiontypes.AuctionRecord{
				QueueTime: queueTime,
			},
		})
	}

	return auctions
}

func BuildTaskAuction(task models.Task, queueTime time.Time) auctiontypes.TaskAuction {
	return auctiontypes.TaskAuction{
		Task: task,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: queueTime,
		},
	}
}

const lucidStack = "lucid64"

var lucidRootFSURL = models.PreloadedRootFS(lucidStack)

var lucidOnlyRootFSProviders = auctiontypes.RootFSProviders{models.PreloadedRootFSScheme: auctiontypes.NewFixedSetRootFSProvider(lucidStack)}

const windowsStack = "windows"

var windowsRootFSURL = models.PreloadedRootFS(windowsStack)

var windowsOnlyRootFSProviders = auctiontypes.RootFSProviders{models.PreloadedRootFSScheme: auctiontypes.NewFixedSetRootFSProvider(windowsStack)}

func BuildCellState(
	zone string,
	memoryMB int,
	diskMB int,
	containers int,
	evacuating bool,
	rootFSProviders auctiontypes.RootFSProviders,
	lrps []auctiontypes.LRP,
) auctiontypes.CellState {
	totalResources := auctiontypes.Resources{
		MemoryMB:   memoryMB,
		DiskMB:     diskMB,
		Containers: containers,
	}

	availableResources := totalResources
	for _, lrp := range lrps {
		availableResources.MemoryMB -= lrp.MemoryMB
		availableResources.DiskMB -= lrp.DiskMB
		availableResources.Containers -= 1
	}

	Expect(availableResources.MemoryMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.DiskMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.Containers).To(BeNumerically(">=", 0), "Check your math!")

	return auctiontypes.CellState{
		RootFSProviders:    rootFSProviders,
		AvailableResources: availableResources,
		TotalResources:     totalResources,
		LRPs:               lrps,
		Zone:               zone,
		Evacuating:         evacuating,
	}
}
