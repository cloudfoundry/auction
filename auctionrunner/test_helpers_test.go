package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/gomega"
)

func BuildLRPStartRequest(processGuid string, indices []uint, stack string, memoryMB, diskMB int) models.LRPStartRequest {
	return models.LRPStartRequest{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			Stack:       stack,
		},
		Indices: indices,
	}
}

func BuildTask(taskGuid, stack string, memoryMB, diskMB int) models.Task {
	return models.Task{
		TaskGuid: taskGuid,
		Stack:    stack,
		MemoryMB: memoryMB,
		DiskMB:   diskMB,
	}
}

func BuildLRPAuction(processGuid string, index int, stack string, memoryMB, diskMB int, queueTime time.Time) auctiontypes.LRPAuction {
	return auctiontypes.LRPAuction{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			Stack:       stack,
		},
		Index: index,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: queueTime,
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

func BuildCellState(memoryMB int, diskMB int, containers int, lrps []auctiontypes.LRP) auctiontypes.CellState {
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

	Ω(availableResources.MemoryMB).Should(BeNumerically(">=", 0), "Check your math!")
	Ω(availableResources.DiskMB).Should(BeNumerically(">=", 0), "Check your math!")
	Ω(availableResources.Containers).Should(BeNumerically(">=", 0), "Check your math!")

	return auctiontypes.CellState{
		Stack:              "lucid64",
		AvailableResources: availableResources,
		TotalResources:     totalResources,
		LRPs:               lrps,
	}
}
