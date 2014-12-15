package auctionrunner_test

import (
	"errors"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cell", func() {
	var client *fakes.FakeSimulationCellRep
	var emptyCell, cell *Cell

	BeforeEach(func() {
		client = &fakes.FakeSimulationCellRep{}
		emptyState := BuildCellState(100, 200, 50, nil)
		emptyCell = NewCell(client, emptyState)

		state := BuildCellState(100, 200, 50, []auctiontypes.LRP{
			{"pg-1", 0, 10, 20},
			{"pg-1", 1, 10, 20},
			{"pg-2", 0, 10, 20},
			{"pg-3", 0, 10, 20},
			{"pg-4", 0, 10, 20},
		})
		cell = NewCell(client, state)
	})

	Describe("ScoreForLRPStartAuction", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", 0, "lucid64", 20, 10)
			smallInstance := BuildLRPStartAuction("pg-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRPStartAuction(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", 0, "lucid64", 10, 20)
			smallInstance := BuildLRPStartAuction("pg-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRPStartAuction(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForLRPStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRPStartAuction("pg-big", 0, "lucid64", 20, 20)

			bigState := BuildCellState(100, 200, 50, nil)
			bigCell := NewCell(client, bigState)

			smallState := BuildCellState(100, 200, 20, nil)
			smallCell := NewCell(client, smallState)

			bigScore, err := bigCell.ScoreForLRPStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := smallCell.ScoreForLRPStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(bigScore).Should(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRPStartAuction("pg-1", 2, "lucid64", 10, 10)
			instanceWithOneMatch := BuildLRPStartAuction("pg-2", 1, "lucid64", 10, 10)
			instanceWithNoMatches := BuildLRPStartAuction("pg-new", 0, "lucid64", 10, 10)

			twoMatchesScore, err := cell.ScoreForLRPStartAuction(instanceWithTwoMatches)
			Ω(err).ShouldNot(HaveOccurred())
			oneMatchesScore, err := cell.ScoreForLRPStartAuction(instanceWithOneMatch)
			Ω(err).ShouldNot(HaveOccurred())
			noMatchesScore, err := cell.ScoreForLRPStartAuction(instanceWithNoMatches)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(noMatchesScore).Should(BeNumerically("<", oneMatchesScore))
			Ω(oneMatchesScore).Should(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRPStartAuction("pg-new", 0, "lucid64", 10000, 10)
					score, err := cell.ScoreForLRPStartAuction(massiveMemoryInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRPStartAuction("pg-new", 0, "lucid64", 10, 10000)
					score, err := cell.ScoreForLRPStartAuction(massiveDiskInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRPStartAuction("pg-new", 0, "lucid64", 10, 10)
					zeroState := BuildCellState(100, 100, 0, nil)
					zeroCell := NewCell(client, zeroState)
					score, err := zeroCell.ScoreForLRPStartAuction(instance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})
		})

		Context("when the LRP doesn't match the stack", func() {
			It("should error", func() {
				nonMatchingInstance := BuildLRPStartAuction("pg-new", 0, ".net", 10, 10)
				score, err := cell.ScoreForLRPStartAuction(nonMatchingInstance)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})
	})

	Describe("ScoreForTask", func() {
		It("factors in memory usage", func() {
			bigTask := BuildTask("tg-big", "lucid64", 20, 10)
			smallTask := BuildTask("tg-small", "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := emptyCell.ScoreForTask(bigTask)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigTask := BuildTask("tg-big", "lucid64", 10, 20)
			smallTask := BuildTask("tg-small", "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := emptyCell.ScoreForTask(bigTask)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForTask(smallTask)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			task := BuildTask("tg-big", "lucid64", 20, 20)

			bigState := BuildCellState(100, 200, 50, nil)
			bigCell := NewCell(client, bigState)

			smallState := BuildCellState(100, 200, 20, nil)
			smallCell := NewCell(client, smallState)

			bigScore, err := bigCell.ScoreForTask(task)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := smallCell.ScoreForTask(task)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(bigScore).Should(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		Context("when the task does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryTask := BuildTask("pg-new", "lucid64", 10000, 10)
					score, err := cell.ScoreForTask(massiveMemoryTask)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskTask := BuildTask("pg-new", "lucid64", 10, 10000)
					score, err := cell.ScoreForTask(massiveDiskTask)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					task := BuildTask("pg-new", "lucid64", 10, 10)
					zeroState := BuildCellState(100, 100, 0, nil)
					zeroCell := NewCell(client, zeroState)
					score, err := zeroCell.ScoreForTask(task)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})
		})

		Context("when the task doesn't match the stack", func() {
			It("should error", func() {
				nonMatchingTask := BuildTask("pg-new", ".net", 10, 10)
				score, err := cell.ScoreForTask(nonMatchingTask)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})
	})

	Describe("StartLRP", func() {
		Context("when there is room for the LRP", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				instance := BuildLRPStartAuction("pg-test", 0, "lucid64", 10, 10)
				instanceToAdd := BuildLRPStartAuction("pg-new", 0, "lucid64", 10, 10)

				initialScore, err := cell.ScoreForLRPStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(cell.StartLRP(instanceToAdd)).Should(Succeed())

				subsequentScore, err := cell.ScoreForLRPStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(initialScore).Should(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the LRP and keep it in mind when handling future requests", func() {
				instance := BuildLRPStartAuction("pg-test", 0, "lucid64", 10, 10)
				instanceWithMatchingProcessGuid := BuildLRPStartAuction("pg-new", 1, "lucid64", 10, 10)
				instanceToAdd := BuildLRPStartAuction("pg-new", 0, "lucid64", 10, 10)

				initialScore, err := cell.ScoreForLRPStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				initialScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRPStartAuction(instanceWithMatchingProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(initialScore).Should(BeNumerically("==", initialScoreForInstanceWithMatchingProcessGuid))

				Ω(cell.StartLRP(instanceToAdd)).Should(Succeed())

				subsequentScore, err := cell.ScoreForLRPStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				subsequentScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRPStartAuction(instanceWithMatchingProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(initialScore).Should(BeNumerically("<", subsequentScore), "the score should have gotten worse")
				Ω(initialScoreForInstanceWithMatchingProcessGuid).Should(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should have gotten worse")

				Ω(subsequentScore).Should(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should be substantially worse for the instance with the matching process guid")
			})
		})

		Context("when there is a stack mismatch", func() {
			It("should error", func() {
				instance := BuildLRPStartAuction("pg-test", 0, ".net", 10, 10)
				err := cell.StartLRP(instance)
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})

		Context("when there is no room for the LRP", func() {
			It("should error", func() {
				instance := BuildLRPStartAuction("pg-test", 0, "lucid64", 10000, 10)
				err := cell.StartLRP(instance)
				Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
			})
		})
	})

	Describe("StartTask", func() {
		Context("when there is room for the task", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				task := BuildTask("tg-test", "lucid64", 10, 10)
				taskToAdd := BuildTask("tg-new", "lucid64", 10, 10)

				initialScore, err := cell.ScoreForTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(cell.StartTask(taskToAdd)).Should(Succeed())

				subsequentScore, err := cell.ScoreForTask(task)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(initialScore).Should(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})
		})

		Context("when there is a stack mismatch", func() {
			It("should error", func() {
				task := BuildTask("tg-test", ".net", 10, 10)
				err := cell.StartTask(task)
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})

		Context("when there is no room for the Task", func() {
			It("should error", func() {
				task := BuildTask("tg-test", "lucid64", 10000, 10)
				err := cell.StartTask(task)
				Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
			})
		})
	})

	Describe("Commit", func() {
		Context("with nothing to commit", func() {
			It("does nothing and returns empty", func() {
				failedWork := cell.Commit()
				Ω(failedWork).Should(BeZero())
				Ω(client.PerformCallCount()).Should(Equal(0))
			})
		})

		Context("with work to commit", func() {
			var instanceToStart models.LRPStartAuction

			BeforeEach(func() {
				instanceToStart = BuildLRPStartAuction("pg-new", 0, "lucid64", 20, 10)

				Ω(cell.StartLRP(instanceToStart)).Should(Succeed())
			})

			It("asks the client to perform", func() {
				cell.Commit()
				Ω(client.PerformCallCount()).Should(Equal(1))
				Ω(client.PerformArgsForCall(0)).Should(Equal(auctiontypes.Work{
					LRPStarts: []models.LRPStartAuction{instanceToStart},
				}))
			})

			Context("when the client returns some failed work", func() {
				It("forwards the failed work", func() {
					failedWork := auctiontypes.Work{
						LRPStarts: []models.LRPStartAuction{instanceToStart},
					}
					client.PerformReturns(failedWork, nil)
					Ω(cell.Commit()).Should(Equal(failedWork))
				})
			})

			Context("when the client returns an error", func() {
				It("does not return any failed work", func() {
					client.PerformReturns(auctiontypes.Work{}, errors.New("boom"))
					Ω(cell.Commit()).Should(BeZero())
				})
			})
		})
	})
})
