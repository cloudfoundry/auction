package auctionrunner_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cell", func() {
	var client *fakes.FakeSimulationCellRep
	var emptyCell, cell *auctionrunner.Cell

	BeforeEach(func() {
		client = &fakes.FakeSimulationCellRep{}
		emptyState := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, nil)
		emptyCell = auctionrunner.NewCell("empty-cell", client, emptyState)

		state := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, []auctiontypes.LRP{
			{"pg-1", 0, 10, 20},
			{"pg-1", 1, 10, 20},
			{"pg-2", 0, 10, 20},
			{"pg-3", 0, 10, 20},
			{"pg-4", 0, 10, 20},
		})
		cell = auctionrunner.NewCell("the-cell", client, state)
	})

	Describe("ScoreForLRPAuction", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRPAuction("pg-big", 0, linuxRootFSURL, 20, 10, time.Now())
			smallInstance := BuildLRPAuction("pg-small", 0, linuxRootFSURL, 10, 10, time.Now())

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRPAuction(bigInstance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRPAuction("pg-big", 0, linuxRootFSURL, 10, 20, time.Now())
			smallInstance := BuildLRPAuction("pg-small", 0, linuxRootFSURL, 10, 10, time.Now())

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRPAuction(bigInstance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForLRPAuction(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRPAuction("pg-big", 0, linuxRootFSURL, 20, 20, time.Now())

			bigState := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, nil)
			bigCell := auctionrunner.NewCell("big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, linuxOnlyRootFSProviders, nil)
			smallCell := auctionrunner.NewCell("small-cell", client, smallState)

			bigScore, err := bigCell.ScoreForLRPAuction(instance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := smallCell.ScoreForLRPAuction(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRPAuction("pg-1", 2, linuxRootFSURL, 10, 10, time.Now())
			instanceWithOneMatch := BuildLRPAuction("pg-2", 1, linuxRootFSURL, 10, 10, time.Now())
			instanceWithNoMatches := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10, 10, time.Now())

			twoMatchesScore, err := cell.ScoreForLRPAuction(instanceWithTwoMatches)
			Expect(err).NotTo(HaveOccurred())
			oneMatchesScore, err := cell.ScoreForLRPAuction(instanceWithOneMatch)
			Expect(err).NotTo(HaveOccurred())
			noMatchesScore, err := cell.ScoreForLRPAuction(instanceWithNoMatches)
			Expect(err).NotTo(HaveOccurred())

			Expect(noMatchesScore).To(BeNumerically("<", oneMatchesScore))
			Expect(oneMatchesScore).To(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10000, 10, time.Now())
					score, err := cell.ScoreForLRPAuction(massiveMemoryInstance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10, 10000, time.Now())
					score, err := cell.ScoreForLRPAuction(massiveDiskInstance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10, 10, time.Now())
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, linuxOnlyRootFSProviders, nil)
					zeroCell := auctionrunner.NewCell("zero-cell", client, zeroState)
					score, err := zeroCell.ScoreForLRPAuction(instance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})
		})

		Describe("matching the RootFS", func() {
			Context("when the cell provides a complex array of RootFSes", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": auctiontypes.ArbitraryRootFSProvider{},
							"arbitrary-2": auctiontypes.ArbitraryRootFSProvider{},
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs with various stack requirements", func() {
					score, err := cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-1", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-2", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-1", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-2", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "arbitrary-1://random-root-fs", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "arbitrary-2://random-root-fs", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					score, err := cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-3", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-3:root-fs-1", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "arbitrary-3://random-root-fs", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})

			Context("when the cell supports a single RootFS", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs requiring the stack supported by the cell", func() {
					score, err := cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-1", 10, 10, time.Now()))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					score, err := cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-2", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-1", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForLRPAuction(BuildLRPAuction("pg", 0, "arbitrary://random-root-fs", 10, 10, time.Now()))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})
		})
	})

	Describe("ScoreForTask", func() {
		It("factors in memory usage", func() {
			bigTask := BuildTask("tg-big", linuxRootFSURL, 20, 10)
			smallTask := BuildTask("tg-small", linuxRootFSURL, 10, 10)

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := emptyCell.ScoreForTask(bigTask)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigTask := BuildTask("tg-big", linuxRootFSURL, 10, 20)
			smallTask := BuildTask("tg-small", linuxRootFSURL, 10, 10)

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := emptyCell.ScoreForTask(bigTask)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForTask(smallTask)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			task := BuildTask("tg-big", linuxRootFSURL, 20, 20)

			bigState := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, nil)
			bigCell := auctionrunner.NewCell("big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, linuxOnlyRootFSProviders, nil)
			smallCell := auctionrunner.NewCell("small-cell", client, smallState)

			bigScore, err := bigCell.ScoreForTask(task)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := smallCell.ScoreForTask(task)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		Context("when the task does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryTask := BuildTask("pg-new", linuxRootFSURL, 10000, 10)
					score, err := cell.ScoreForTask(massiveMemoryTask)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskTask := BuildTask("pg-new", linuxRootFSURL, 10, 10000)
					score, err := cell.ScoreForTask(massiveDiskTask)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					task := BuildTask("pg-new", linuxRootFSURL, 10, 10)
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, linuxOnlyRootFSProviders, nil)
					zeroCell := auctionrunner.NewCell("zero-cell", client, zeroState)
					score, err := zeroCell.ScoreForTask(task)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
				})
			})
		})

		Describe("matching the RootFS", func() {
			Context("when the cell provides a complex array of RootFSes", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": auctiontypes.ArbitraryRootFSProvider{},
							"arbitrary-2": auctiontypes.ArbitraryRootFSProvider{},
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks with various stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "fixed-set-2:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "arbitrary-1://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "arbitrary-2://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "fixed-set-1:root-fs-3", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "fixed-set-3:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "arbitrary-3://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})

			Context("when the cell supports a single RootFS", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks requiring the stack supported by the cell", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "arbitrary://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})
		})
	})

	Describe("ReserveLRP", func() {
		Context("when there is room for the LRP", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				instance := BuildLRPAuction("pg-test", 0, linuxRootFSURL, 10, 10, time.Now())
				instanceToAdd := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10, 10, time.Now())

				initialScore, err := cell.ScoreForLRPAuction(instance)
				Expect(err).NotTo(HaveOccurred())

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := cell.ScoreForLRPAuction(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the LRP and keep it in mind when handling future requests", func() {
				instance := BuildLRPAuction("pg-test", 0, linuxRootFSURL, 10, 10, time.Now())
				instanceWithMatchingProcessGuid := BuildLRPAuction("pg-new", 1, linuxRootFSURL, 10, 10, time.Now())
				instanceToAdd := BuildLRPAuction("pg-new", 0, linuxRootFSURL, 10, 10, time.Now())

				initialScore, err := cell.ScoreForLRPAuction(instance)
				Expect(err).NotTo(HaveOccurred())

				initialScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRPAuction(instanceWithMatchingProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("==", initialScoreForInstanceWithMatchingProcessGuid))

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := cell.ScoreForLRPAuction(instance)
				Expect(err).NotTo(HaveOccurred())

				subsequentScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRPAuction(instanceWithMatchingProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
				Expect(initialScoreForInstanceWithMatchingProcessGuid).To(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should have gotten worse")

				Expect(subsequentScore).To(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should be substantially worse for the instance with the matching process guid")
			})
		})

		Describe("matching the RootFS", func() {
			Context("when the cell provides a complex array of RootFSes", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": auctiontypes.ArbitraryRootFSProvider{},
							"arbitrary-2": auctiontypes.ArbitraryRootFSProvider{},
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs with various stack requirements", func() {
					err := cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-1", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-2", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-1", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-2", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "arbitrary-1://random-root-fs", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "arbitrary-2://random-root-fs", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					err := cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-3", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-3:root-fs-1", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "arbitrary-3://random-root-fs", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})

			Context("when the cell supports a single RootFS", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs requiring the stack supported by the cell", func() {
					err := cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-1", 10, 10, time.Now()))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					err := cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-1:root-fs-2", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "fixed-set-2:root-fs-1", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveLRP(BuildLRPAuction("pg", 0, "arbitrary://random-root-fs", 10, 10, time.Now()))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})
		})

		Context("when there is no room for the LRP", func() {
			It("should error", func() {
				instance := BuildLRPAuction("pg-test", 0, linuxRootFSURL, 10000, 10, time.Now())
				err := cell.ReserveLRP(instance)
				Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
			})
		})
	})

	Describe("ReserveTask", func() {
		Context("when there is room for the task", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				task := BuildTask("tg-test", linuxRootFSURL, 10, 10)
				taskToAdd := BuildTask("tg-new", linuxRootFSURL, 10, 10)

				initialScore, err := cell.ScoreForTask(task)
				Expect(err).NotTo(HaveOccurred())

				Expect(cell.ReserveTask(taskToAdd)).To(Succeed())

				subsequentScore, err := cell.ScoreForTask(task)
				Expect(err).NotTo(HaveOccurred())
				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})
		})

		Describe("matching the RootFS", func() {
			Context("when the cell provides a complex array of RootFSes", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": auctiontypes.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": auctiontypes.ArbitraryRootFSProvider{},
							"arbitrary-2": auctiontypes.ArbitraryRootFSProvider{},
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks with various stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "fixed-set-2:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "arbitrary-1://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "arbitrary-2://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "fixed-set-1:root-fs-3", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveTask(BuildTask("task-guid", "fixed-set-3:root-fs-1", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveTask(BuildTask("task-guid", "arbitrary-3://random-root-fs", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})

			Context("when the cell supports a single RootFS", func() {
				BeforeEach(func() {
					state := BuildCellState(
						"the-zone",
						100,
						100,
						100,
						false,
						auctiontypes.RootFSProviders{
							"fixed-set-1": auctiontypes.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]auctiontypes.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks requiring the stack supported by the cell", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveTask(BuildTask("task-guid", "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))

					err = cell.ReserveTask(BuildTask("task-guid", "arbitrary://random-root-fs", 10, 10))
					Expect(err).To(MatchError(auctiontypes.ErrorCellMismatch))
				})
			})
		})

		Context("when there is no room for the Task", func() {
			It("should error", func() {
				task := BuildTask("tg-test", linuxRootFSURL, 10000, 10)
				err := cell.ReserveTask(task)
				Expect(err).To(MatchError(auctiontypes.ErrorInsufficientResources))
			})
		})
	})

	Describe("Commit", func() {
		Context("with nothing to commit", func() {
			It("does nothing and returns empty", func() {
				failedWork := cell.Commit()
				Expect(failedWork).To(BeZero())
				Expect(client.PerformCallCount()).To(Equal(0))
			})
		})

		Context("with work to commit", func() {
			var lrpAuction auctiontypes.LRPAuction

			BeforeEach(func() {
				lrpAuction = BuildLRPAuction("pg-new", 0, linuxRootFSURL, 20, 10, time.Now())

				Expect(cell.ReserveLRP(lrpAuction)).To(Succeed())
			})

			It("asks the client to perform", func() {
				cell.Commit()
				Expect(client.PerformCallCount()).To(Equal(1))
				Expect(client.PerformArgsForCall(0)).To(Equal(auctiontypes.Work{
					LRPs: []auctiontypes.LRPAuction{lrpAuction},
				}))

			})

			Context("when the client returns some failed work", func() {
				It("forwards the failed work", func() {
					failedWork := auctiontypes.Work{
						LRPs: []auctiontypes.LRPAuction{lrpAuction},
					}
					client.PerformReturns(failedWork, nil)
					Expect(cell.Commit()).To(Equal(failedWork))
				})
			})

			Context("when the client returns an error", func() {
				It("does not return any failed work", func() {
					client.PerformReturns(auctiontypes.Work{}, errors.New("boom"))
					Expect(cell.Commit()).To(BeZero())
				})
			})
		})
	})
})
