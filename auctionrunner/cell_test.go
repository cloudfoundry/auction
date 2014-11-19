package auctionrunner_test

import (
	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func BuildLRPStartAuction(processGuid string, instanceGuid string, index int, stack string, memoryMB int, diskMB int) models.LRPStartAuction {
	return models.LRPStartAuction{
		DesiredLRP: models.DesiredLRP{
			ProcessGuid: processGuid,
			MemoryMB:    memoryMB,
			DiskMB:      diskMB,
			Stack:       stack,
		},
		InstanceGuid: instanceGuid,
		Index:        index,
	}
}

func BuildRepState(memoryMB int, diskMB int, containers int, lrps []auctiontypes.LRP) auctiontypes.RepState {
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

	return auctiontypes.RepState{
		Stack:              "lucid64",
		AvailableResources: availableResources,
		TotalResources:     totalResources,
		LRPs:               lrps,
	}
}

var _ = Describe("Cell", func() {
	var emptyClient, client *fakes.FakeSimulationAuctionRep
	var emptyCell, cell *Cell

	BeforeEach(func() {
		emptyClient = &fakes.FakeSimulationAuctionRep{}
		emptyState := BuildRepState(100, 200, 50, nil)
		emptyCell = NewCell(emptyClient, emptyState)

		client = &fakes.FakeSimulationAuctionRep{}
		state := BuildRepState(100, 200, 50, []auctiontypes.LRP{
			{"pg-1", "ig-1", 0, 10, 20},
			{"pg-1", "ig-2", 1, 10, 20},
			{"pg-2", "ig-3", 0, 10, 20},
			{"pg-3", "ig-4", 0, 10, 20},
			{"pg-4", "ig-5", 0, 10, 20},
		})
		cell = NewCell(client, state)
	})

	Describe("ScoreToStartLRP", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 20, 10)
			smallInstance := BuildLRPStartAuction("pg-small", "ig-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreToStartLRP(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 10, 20)
			smallInstance := BuildLRPStartAuction("pg-small", "ig-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreToStartLRP(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreToStartLRP(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 20, 20)

			bigState := BuildRepState(100, 200, 50, nil)
			bigCell := NewCell(client, bigState)

			smallState := BuildRepState(100, 200, 20, nil)
			smallCell := NewCell(client, smallState)

			bigScore, err := bigCell.ScoreToStartLRP(instance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := smallCell.ScoreToStartLRP(instance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(bigScore).Should(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRPStartAuction("pg-1", "ig-new", 2, "lucid64", 10, 10)
			instanceWithOneMatch := BuildLRPStartAuction("pg-2", "ig-new", 1, "lucid64", 10, 10)
			instanceWithNoMatches := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)

			twoMatchesScore, err := cell.ScoreToStartLRP(instanceWithTwoMatches)
			Ω(err).ShouldNot(HaveOccurred())
			oneMatchesScore, err := cell.ScoreToStartLRP(instanceWithOneMatch)
			Ω(err).ShouldNot(HaveOccurred())
			noMatchesScore, err := cell.ScoreToStartLRP(instanceWithNoMatches)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(noMatchesScore).Should(BeNumerically("<", oneMatchesScore))
			Ω(oneMatchesScore).Should(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10000, 10)
					score, err := cell.ScoreToStartLRP(massiveMemoryInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(InsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10000)
					score, err := cell.ScoreToStartLRP(massiveDiskInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(InsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)
					zeroState := BuildRepState(100, 100, 0, nil)
					zeroCell := NewCell(client, zeroState)
					score, err := zeroCell.ScoreToStartLRP(instance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(InsufficientResources))
				})
			})
		})

		Context("when the LRP doesn't match the stack", func() {
			It("should error", func() {
				nonMatchingInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, ".net", 10, 10)
				score, err := cell.ScoreToStartLRP(nonMatchingInstance)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(StackMismatch))
			})
		})
	})
})
