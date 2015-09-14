package auctionrunner_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry-incubator/rep/repfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cell", func() {
	var client *repfakes.FakeSimClient
	var emptyCell, cell *auctionrunner.Cell

	BeforeEach(func() {
		client = &repfakes.FakeSimClient{}
		emptyState := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, nil)
		emptyCell = auctionrunner.NewCell("empty-cell", client, emptyState)

		state := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, []rep.LRP{
			*BuildLRP("pg-1", "domain", 0, linuxRootFSURL, 10, 20),
			*BuildLRP("pg-1", "domain", 1, linuxRootFSURL, 10, 20),
			*BuildLRP("pg-2", "domain", 0, linuxRootFSURL, 10, 20),
			*BuildLRP("pg-3", "domain", 0, linuxRootFSURL, 10, 20),
			*BuildLRP("pg-4", "domain", 0, linuxRootFSURL, 10, 20),
		})
		cell = auctionrunner.NewCell("the-cell", client, state)
	})

	Describe("ScoreForLRP", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 10)
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRP(bigInstance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 10, 20)
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10)

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := emptyCell.ScoreForLRP(bigInstance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := emptyCell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := emptyCell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			score, err := cell.ScoreForLRP(smallInstance)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 20)

			bigState := BuildCellState("the-zone", 100, 200, 50, false, linuxOnlyRootFSProviders, nil)
			bigCell := auctionrunner.NewCell("big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, linuxOnlyRootFSProviders, nil)
			smallCell := auctionrunner.NewCell("small-cell", client, smallState)

			bigScore, err := bigCell.ScoreForLRP(instance)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := smallCell.ScoreForLRP(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRP("pg-1", "domain", 2, linuxRootFSURL, 10, 10)
			instanceWithOneMatch := BuildLRP("pg-2", "domain", 1, linuxRootFSURL, 10, 10)
			instanceWithNoMatches := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10)

			twoMatchesScore, err := cell.ScoreForLRP(instanceWithTwoMatches)
			Expect(err).NotTo(HaveOccurred())
			oneMatchesScore, err := cell.ScoreForLRP(instanceWithOneMatch)
			Expect(err).NotTo(HaveOccurred())
			noMatchesScore, err := cell.ScoreForLRP(instanceWithNoMatches)
			Expect(err).NotTo(HaveOccurred())

			Expect(noMatchesScore).To(BeNumerically("<", oneMatchesScore))
			Expect(oneMatchesScore).To(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10000, 10)
					score, err := cell.ScoreForLRP(massiveMemoryInstance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10000)
					score, err := cell.ScoreForLRP(massiveDiskInstance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10)
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, linuxOnlyRootFSProviders, nil)
					zeroCell := auctionrunner.NewCell("zero-cell", client, zeroState)
					score, err := zeroCell.ScoreForLRP(instance)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": rep.ArbitraryRootFSProvider{},
							"arbitrary-2": rep.ArbitraryRootFSProvider{},
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs with various stack requirements", func() {
					score, err := cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "arbitrary-1://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "arbitrary-2://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					score, err := cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-3", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-3:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "arbitrary-3://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs requiring the stack supported by the cell", func() {
					score, err := cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					score, err := cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForLRP(BuildLRP("pg", "domain", 0, "arbitrary://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
				})
			})
		})
	})

	Describe("ScoreForTask", func() {
		It("factors in memory usage", func() {
			bigTask := BuildTask("tg-big", "domain", linuxRootFSURL, 20, 10)
			smallTask := BuildTask("tg-small", "domain", linuxRootFSURL, 10, 10)

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
			bigTask := BuildTask("tg-big", "domain", linuxRootFSURL, 10, 20)
			smallTask := BuildTask("tg-small", "domain", linuxRootFSURL, 10, 10)

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
			task := BuildTask("tg-big", "domain", linuxRootFSURL, 20, 20)

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
					massiveMemoryTask := BuildTask("pg-new", "domain", linuxRootFSURL, 10000, 10)
					score, err := cell.ScoreForTask(massiveMemoryTask)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskTask := BuildTask("pg-new", "domain", linuxRootFSURL, 10, 10000)
					score, err := cell.ScoreForTask(massiveDiskTask)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					task := BuildTask("pg-new", "domain", linuxRootFSURL, 10, 10)
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, linuxOnlyRootFSProviders, nil)
					zeroCell := auctionrunner.NewCell("zero-cell", client, zeroState)
					score, err := zeroCell.ScoreForTask(task)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorInsufficientResources))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": rep.ArbitraryRootFSProvider{},
							"arbitrary-2": rep.ArbitraryRootFSProvider{},
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks with various stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-2", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "arbitrary-1://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "arbitrary-2://random-root-fs", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-3", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-3:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "arbitrary-3://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks requiring the stack supported by the cell", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-1", 10, 10))
					Expect(score).To(BeNumerically(">", 0))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					score, err := cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-2", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-1", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					score, err = cell.ScoreForTask(BuildTask("task-guid", "domain", "arbitrary://random-root-fs", 10, 10))
					Expect(score).To(BeZero())
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
				})
			})
		})
	})

	Describe("ReserveLRP", func() {
		Context("when there is room for the LRP", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10, 10)
				instanceToAdd := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10)

				initialScore, err := cell.ScoreForLRP(instance)
				Expect(err).NotTo(HaveOccurred())

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := cell.ScoreForLRP(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the LRP and keep it in mind when handling future requests", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10, 10)
				instanceWithMatchingProcessGuid := BuildLRP("pg-new", "domain", 1, linuxRootFSURL, 10, 10)
				instanceToAdd := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10)

				initialScore, err := cell.ScoreForLRP(instance)
				Expect(err).NotTo(HaveOccurred())

				initialScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRP(instanceWithMatchingProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("==", initialScoreForInstanceWithMatchingProcessGuid))

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := cell.ScoreForLRP(instance)
				Expect(err).NotTo(HaveOccurred())

				subsequentScoreForInstanceWithMatchingProcessGuid, err := cell.ScoreForLRP(instanceWithMatchingProcessGuid)
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": rep.ArbitraryRootFSProvider{},
							"arbitrary-2": rep.ArbitraryRootFSProvider{},
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs with various stack requirements", func() {
					err := cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "arbitrary-1://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "arbitrary-2://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					err := cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-3", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-3:root-fs-1", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "arbitrary-3://random-root-fs", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support LRPs requiring the stack supported by the cell", func() {
					err := cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for LRPs with unsupported stack requirements", func() {
					err := cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveLRP(BuildLRP("pg", "domain", 0, "arbitrary://random-root-fs", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
				})
			})
		})

		Context("when there is no room for the LRP", func() {
			It("should error", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10000, 10)
				err := cell.ReserveLRP(instance)
				Expect(err).To(MatchError(rep.ErrorInsufficientResources))
			})
		})
	})

	Describe("ReserveTask", func() {
		Context("when there is room for the task", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				task := BuildTask("tg-test", "domain", linuxRootFSURL, 10, 10)
				taskToAdd := BuildTask("tg-new", "domain", linuxRootFSURL, 10, 10)

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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"fixed-set-2": rep.NewFixedSetRootFSProvider("root-fs-1", "root-fs-2"),
							"arbitrary-1": rep.ArbitraryRootFSProvider{},
							"arbitrary-2": rep.ArbitraryRootFSProvider{},
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks with various stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-2", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "arbitrary-1://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "arbitrary-2://random-root-fs", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-3", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-3:root-fs-1", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "arbitrary-3://random-root-fs", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
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
						rep.RootFSProviders{
							"fixed-set-1": rep.NewFixedSetRootFSProvider("root-fs-1"),
						},
						[]rep.LRP{},
					)
					cell = auctionrunner.NewCell("the-cell", client, state)
				})

				It("should support Tasks requiring the stack supported by the cell", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-1", 10, 10))
					Expect(err).NotTo(HaveOccurred())
				})

				It("should error for Tasks with unsupported stack requirements", func() {
					err := cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-1:root-fs-2", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "fixed-set-2:root-fs-1", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))

					err = cell.ReserveTask(BuildTask("task-guid", "domain", "arbitrary://random-root-fs", 10, 10))
					Expect(err).To(MatchError(rep.ErrorIncompatibleRootfs))
				})
			})
		})

		Context("when there is no room for the Task", func() {
			It("should error", func() {
				task := BuildTask("tg-test", "domain", linuxRootFSURL, 10000, 10)
				err := cell.ReserveTask(task)
				Expect(err).To(MatchError(rep.ErrorInsufficientResources))
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
			var lrp rep.LRP

			BeforeEach(func() {
				lrp = *BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 20, 10)
				Expect(cell.ReserveLRP(&lrp)).To(Succeed())
			})

			It("asks the client to perform", func() {
				cell.Commit()
				Expect(client.PerformCallCount()).To(Equal(1))
				Expect(client.PerformArgsForCall(0)).To(Equal(rep.Work{
					LRPs: []rep.LRP{lrp},
				}))

			})

			Context("when the client returns some failed work", func() {
				It("forwards the failed work", func() {
					failedWork := rep.Work{
						LRPs: []rep.LRP{lrp},
					}
					client.PerformReturns(failedWork, nil)
					Expect(cell.Commit()).To(Equal(failedWork))
				})
			})

			Context("when the client returns an error", func() {
				It("does not return any failed work", func() {
					client.PerformReturns(rep.Work{}, errors.New("boom"))
					Expect(cell.Commit()).To(BeZero())
				})
			})
		})
	})
})
