package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/gomega"
)

func BuildLRPStart(processGuid string, index int, stack string, memoryMB, diskMB int) models.LRPStart {
	return models.LRPStart{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			Stack:       stack,
		},
		Index: index,
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

func BuildStartAuction(start models.LRPStart, queueTime time.Time) auctiontypes.LRPStartAuction {
	return auctiontypes.LRPStartAuction{
		LRPStart: start,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: queueTime,
		},
	}
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
