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
	var client *fakes.FakeSimulationAuctionRep
	var emptyCell, cell *Cell

	BeforeEach(func() {
		client = &fakes.FakeSimulationAuctionRep{}
		emptyState := BuildRepState(100, 200, 50, nil)
		emptyCell = NewCell(client, emptyState)

		state := BuildRepState(100, 200, 50, []auctiontypes.LRP{
			{"pg-1", "ig-1", 0, 10, 20},
			{"pg-1", "ig-2", 1, 10, 20},
			{"pg-2", "ig-3", 0, 10, 20},
			{"pg-3", "ig-4", 0, 10, 20},
			{"pg-4", "ig-5", 0, 10, 20},
		})
		cell = NewCell(client, state)
	})

	Describe("ScoreForStartAuction", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 20, 10)
			smallInstance := BuildLRPStartAuction("pg-small", "ig-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForStartAuction(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 10, 20)
			smallInstance := BuildLRPStartAuction("pg-small", "ig-small", 0, "lucid64", 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForStartAuction(bigInstance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := emptyCell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(smallScore).Should(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			score, err := cell.ScoreForStartAuction(smallInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(emptyScore).Should(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRPStartAuction("pg-big", "ig-big", 0, "lucid64", 20, 20)

			bigState := BuildRepState(100, 200, 50, nil)
			bigCell := NewCell(client, bigState)

			smallState := BuildRepState(100, 200, 20, nil)
			smallCell := NewCell(client, smallState)

			bigScore, err := bigCell.ScoreForStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())
			smallScore, err := smallCell.ScoreForStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(bigScore).Should(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRPStartAuction("pg-1", "ig-new", 2, "lucid64", 10, 10)
			instanceWithOneMatch := BuildLRPStartAuction("pg-2", "ig-new", 1, "lucid64", 10, 10)
			instanceWithNoMatches := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)

			twoMatchesScore, err := cell.ScoreForStartAuction(instanceWithTwoMatches)
			Ω(err).ShouldNot(HaveOccurred())
			oneMatchesScore, err := cell.ScoreForStartAuction(instanceWithOneMatch)
			Ω(err).ShouldNot(HaveOccurred())
			noMatchesScore, err := cell.ScoreForStartAuction(instanceWithNoMatches)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(noMatchesScore).Should(BeNumerically("<", oneMatchesScore))
			Ω(oneMatchesScore).Should(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10000, 10)
					score, err := cell.ScoreForStartAuction(massiveMemoryInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10000)
					score, err := cell.ScoreForStartAuction(massiveDiskInstance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)
					zeroState := BuildRepState(100, 100, 0, nil)
					zeroCell := NewCell(client, zeroState)
					score, err := zeroCell.ScoreForStartAuction(instance)
					Ω(score).Should(BeZero())
					Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})
		})

		Context("when the LRP doesn't match the stack", func() {
			It("should error", func() {
				nonMatchingInstance := BuildLRPStartAuction("pg-new", "ig-new", 0, ".net", 10, 10)
				score, err := cell.ScoreForStartAuction(nonMatchingInstance)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})
	})

	Describe("ScoreForStopAuction", func() {
		var stopAuction models.LRPStopAuction
		BeforeEach(func() {
			stopAuction = BuildLRPStopAuction("pg-1", 1)
		})

		It("factors in memory constraints", func() {
			cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-1", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			cellB := NewCell(client, BuildRepState(50, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-2", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			scoreA, instancesA, err := cellA.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			scoreB, instancesB, err := cellB.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
			Ω(instancesA).Should(ConsistOf("ig-1"))
			Ω(instancesB).Should(ConsistOf("ig-2"))
		})

		It("factors in disk constraints", func() {
			cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-1", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			cellB := NewCell(client, BuildRepState(100, 100, 50, []auctiontypes.LRP{
				{"pg-1", "ig-2", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			scoreA, instancesA, err := cellA.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			scoreB, instancesB, err := cellB.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
			Ω(instancesA).Should(ConsistOf("ig-1"))
			Ω(instancesB).Should(ConsistOf("ig-2"))
		})

		It("factors in container constraints", func() {
			cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-1", 1, 10, 20},
			}))

			cellB := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-2", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			scoreA, instancesA, err := cellA.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			scoreB, instancesB, err := cellB.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
			Ω(instancesA).Should(ConsistOf("ig-1"))
			Ω(instancesB).Should(ConsistOf("ig-2"))
		})

		It("factors in colocated process guids", func() {
			cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-1", 1, 10, 20},
				{"pg-other", "ig-other", 0, 10, 20},
			}))

			cellB := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
				{"pg-1", "ig-2", 1, 10, 20},
				{"pg-1", "ig-3", 2, 10, 20},
			}))

			scoreA, instancesA, err := cellA.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			scoreB, instancesB, err := cellB.ScoreForStopAuction(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
			Ω(instancesA).Should(ConsistOf("ig-1"))
			Ω(instancesB).Should(ConsistOf("ig-2"))
		})

		Context("with multiple instances on an individual cell", func() {
			It("returns an array of instance guids that can be stopped", func() {
				cell := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
					{"pg-1", "ig-1", 1, 10, 20},
					{"pg-1", "ig-2", 1, 10, 20},
				}))

				_, instances, err := cell.ScoreForStopAuction(stopAuction)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(instances).Should(ConsistOf("ig-1", "ig-2"))
			})

			Context("when it makes sense to preserve an instance on a cell with multiple instances", func() {
				It("should give that cell a lower score", func() {
					cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-1", 1, 10, 20},
						{"pg-1", "ig-2", 1, 10, 20},
					}))

					cellB := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-3", 1, 10, 20},
						{"pg-other", "ig-other", 0, 10, 20},
						{"pg-yet-another", "ig-yet-another", 1, 10, 20},
					}))

					scoreA, _, err := cellA.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					scoreB, _, err := cellB.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
				})
			})

			Context("when it makes sense to preserve an instance on a cell with just one instance", func() {
				It("should give that cell a lower score", func() {
					cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-1", 1, 10, 20},
					}))

					cellB := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-2", 1, 10, 20},
						{"pg-1", "ig-3", 1, 10, 20},
						{"pg-yet-another", "ig-yet-another", 1, 10, 20},
					}))

					scoreA, _, err := cellA.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					scoreB, _, err := cellB.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreA).Should(BeNumerically("<", scoreB), "it's preferable to preserve the instance on A")
				})
			})

			Context("when it's a crapshoot", func() {
				It("should give both cells the same score", func() {
					cellA := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-1", 1, 10, 20},
						{"pg-yet-another", "ig-yet-another", 1, 10, 20},
					}))

					cellB := NewCell(client, BuildRepState(100, 200, 50, []auctiontypes.LRP{
						{"pg-1", "ig-2", 1, 10, 20},
						{"pg-1", "ig-3", 1, 10, 20},
						{"pg-yet-another", "ig-yet-another", 1, 10, 20},
					}))

					scoreA, _, err := cellA.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					scoreB, _, err := cellB.ScoreForStopAuction(stopAuction)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreA).Should(BeNumerically("==", scoreB))
				})
			})
		})

		Context("when there are no matching process guids", func() {
			It("should return an error", func() {
				noneMatchingAuction := BuildLRPStopAuction("pg-none", 0)
				score, instanceGuids, err := cell.ScoreForStopAuction(noneMatchingAuction)
				Ω(score).Should(BeZero())
				Ω(instanceGuids).Should(BeEmpty())
				Ω(err).Should(MatchError(auctiontypes.ErrorNothingToStop))
			})
		})

		Context("when there are no matching indices", func() {
			It("should return an error", func() {
				noneMatchingAuction := BuildLRPStopAuction("pg-1", 17)
				score, instanceGuids, err := cell.ScoreForStopAuction(noneMatchingAuction)
				Ω(score).Should(BeZero())
				Ω(instanceGuids).Should(BeEmpty())
				Ω(err).Should(MatchError(auctiontypes.ErrorNothingToStop))
			})
		})
	})

	Describe("StartLRP", func() {
		Context("when there is room for the LRP", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				instance := BuildLRPStartAuction("pg-test", "ig-test", 0, "lucid64", 10, 10)
				instanceToAdd := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)

				initialScore, err := cell.ScoreForStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(cell.StartLRP(instanceToAdd)).Should(Succeed())

				subsequentScore, err := cell.ScoreForStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(initialScore).Should(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the LRP and keep it in mind when handling future requests", func() {
				instance := BuildLRPStartAuction("pg-test", "ig-test", 0, "lucid64", 10, 10)
				instanceWithMatchingProcessGuid := BuildLRPStartAuction("pg-new", "ig-new-2", 1, "lucid64", 10, 10)
				instanceToAdd := BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 10, 10)

				initialScore, err := cell.ScoreForStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				initialScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForStartAuction(instanceWithMatchingProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(initialScore).Should(BeNumerically("==", initialScoreForInstanceWithMatchingProcessGuid))

				Ω(cell.StartLRP(instanceToAdd)).Should(Succeed())

				subsequentScore, err := cell.ScoreForStartAuction(instance)
				Ω(err).ShouldNot(HaveOccurred())

				subsequentScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForStartAuction(instanceWithMatchingProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(initialScore).Should(BeNumerically("<", subsequentScore), "the score should have gotten worse")
				Ω(initialScoreForInstanceWithMatchingProcessGuid).Should(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should have gotten worse")

				Ω(subsequentScore).Should(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should be substantially worse for the instance with the matching process guid")
			})
		})

		Context("when there is a stack mismatch", func() {
			It("should error", func() {
				instance := BuildLRPStartAuction("pg-test", "ig-test", 0, ".net", 10, 10)
				err := cell.StartLRP(instance)
				Ω(err).Should(MatchError(auctiontypes.ErrorStackMismatch))
			})
		})

		Context("when there is no room for the LRP", func() {
			It("should error", func() {
				instance := BuildLRPStartAuction("pg-test", "ig-test", 0, "lucid64", 10000, 10)
				err := cell.StartLRP(instance)
				Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
			})
		})
	})

	Describe("StopLRP", func() {
		It("removes the LRP and keep the fact in mind when handling future requests", func() {
			instance := BuildLRPStartAuction("pg-test", "ig-test", 0, "lucid64", 10, 10)
			instanceToStop := models.StopLRPInstance{
				ProcessGuid:  "pg-1",
				InstanceGuid: "ig-2",
				Index:        1,
			}

			initialScore, err := cell.ScoreForStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(cell.StopLRP(instanceToStop)).Should(Succeed())

			subsequentScore, err := cell.ScoreForStartAuction(instance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(initialScore).Should(BeNumerically(">", subsequentScore), "the score should have gotten better")
		})

		It("removes the LRP, making it impossible to remove again", func() {
			instanceToStop := models.StopLRPInstance{
				ProcessGuid:  "pg-1",
				InstanceGuid: "ig-2",
				Index:        1,
			}

			Ω(cell.StopLRP(instanceToStop)).Should(Succeed())
			Ω(cell.StopLRP(instanceToStop)).ShouldNot(Succeed())
		})

		Context("when the lrp is not present", func() {
			It("returns an error", func() {
				Because := By

				Because("of a mismatched process guid")
				instanceToStop := models.StopLRPInstance{
					ProcessGuid:  "pg-0",
					InstanceGuid: "ig-2",
					Index:        1,
				}
				err := cell.StopLRP(instanceToStop)
				Ω(err).Should(MatchError(auctiontypes.ErrorNothingToStop))

				Because("of a mismatched instance guid")
				instanceToStop = models.StopLRPInstance{
					ProcessGuid:  "pg-1",
					InstanceGuid: "ig-3",
					Index:        1,
				}
				err = cell.StopLRP(instanceToStop)
				Ω(err).Should(MatchError(auctiontypes.ErrorNothingToStop))

				Because("of a mismatched index")
				instanceToStop = models.StopLRPInstance{
					ProcessGuid:  "pg-1",
					InstanceGuid: "ig-2",
					Index:        0,
				}
				err = cell.StopLRP(instanceToStop)
				Ω(err).Should(MatchError(auctiontypes.ErrorNothingToStop))
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
			var instanceToStop models.StopLRPInstance

			BeforeEach(func() {
				instanceToStart = BuildLRPStartAuction("pg-new", "ig-new", 0, "lucid64", 20, 10)
				instanceToStop = models.StopLRPInstance{
					ProcessGuid:  "pg-1",
					InstanceGuid: "ig-2",
					Index:        1,
				}

				Ω(cell.StartLRP(instanceToStart)).Should(Succeed())
				Ω(cell.StopLRP(instanceToStop)).Should(Succeed())
			})

			It("asks the client to perform", func() {
				cell.Commit()
				Ω(client.PerformCallCount()).Should(Equal(1))
				Ω(client.PerformArgsForCall(0)).Should(Equal(auctiontypes.Work{
					Starts: []models.LRPStartAuction{instanceToStart},
					Stops:  []models.StopLRPInstance{instanceToStop},
				}))
			})

			Context("when the client returns some failed work", func() {
				It("forwards the failed work", func() {
					failedWork := auctiontypes.Work{
						Stops: []models.StopLRPInstance{instanceToStop},
					}
					client.PerformReturns(failedWork, nil)
					Ω(cell.Commit()).Should(Equal(failedWork))
				})
			})

			Context("when the client returns an error", func() {
				It("returns all work as failed work", func() {
					client.PerformReturns(auctiontypes.Work{}, errors.New("boom"))
					Ω(cell.Commit()).Should(Equal(auctiontypes.Work{
						Starts: []models.LRPStartAuction{instanceToStart},
						Stops:  []models.StopLRPInstance{instanceToStop},
					}))
				})
			})
		})
	})
})
