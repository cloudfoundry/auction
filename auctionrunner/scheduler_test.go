package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/gunk/workpool"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scheduler", func() {
	var clients map[string]*fakes.FakeSimulationCellRep
	var cells map[string]*Cell
	var timeProvider *faketimeprovider.FakeTimeProvider
	var workPool *workpool.WorkPool
	var results auctiontypes.AuctionResults
	var metricSender *fake.FakeMetricSender

	BeforeEach(func() {
		metricSender = fake.NewFakeMetricSender()
		metrics.Initialize(metricSender)

		timeProvider = faketimeprovider.New(time.Now())
		workPool = workpool.NewWorkPool(5)

		clients = map[string]*fakes.FakeSimulationCellRep{}
		cells = map[string]*Cell{}
	})

	AfterEach(func() {
		workPool.Stop()
	})

	itBumpsTheStartedCounters := func(lrps int, tasks int) {
		By("successful starts", func() {
			Ω(metricSender.GetCounter("AuctioneerLRPAuctionsStarted")).Should(BeNumerically("==", lrps))
			Ω(metricSender.GetCounter("AuctioneerTaskAuctionsStarted")).Should(BeNumerically("==", tasks))
		})
	}

	Context("when the cells are empty", func() {
		It("immediately returns everything as having failed, incrementing the attempt number", func() {
			startAuction := BuildLRPAuction("pg-7", 0, "lucid64", 10, 10, timeProvider.Now())

			taskAuction := BuildTaskAuction(BuildTask("tg-1", "lucid64", 0, 0), timeProvider.Now())

			auctionRequest := auctiontypes.AuctionRequest{
				LRPs:  []auctiontypes.LRPAuction{startAuction},
				Tasks: []auctiontypes.TaskAuction{taskAuction},
			}

			By("no auctions are marked successful")
			results := Schedule(workPool, map[string]*Cell{}, timeProvider, auctionRequest)
			Ω(results.SuccessfulLRPs).Should(BeEmpty())
			Ω(results.SuccessfulTasks).Should(BeEmpty())

			By("all lrp starts are marked failed, and their attempts are incremented")
			Ω(results.FailedLRPs).Should(HaveLen(1))
			failedLRPStart := results.FailedLRPs[0]
			Ω(failedLRPStart.Identifier()).Should(Equal(startAuction.Identifier()))
			Ω(failedLRPStart.Attempts).Should(Equal(startAuction.Attempts + 1))

			By("all tasks are marked failed, and their attempts are incremented")
			Ω(results.FailedTasks).Should(HaveLen(1))
			failedTask := results.FailedTasks[0]
			Ω(failedTask.Identifier()).Should(Equal(taskAuction.Identifier()))
			Ω(failedTask.Attempts).Should(Equal(taskAuction.Attempts + 1))

			itBumpsTheStartedCounters(0, 0)
		})

	})

	Describe("handling start auctions", func() {
		var startAuction auctiontypes.LRPAuction

		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", 0, 10, 10},
				{"pg-2", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-3", 0, 10, 10},
			}))

			startAuction = BuildLRPAuction("pg-4", 0, "lucid64", 10, 10, timeProvider.Now())
			timeProvider.Increment(time.Minute)
		})

		Context("when it picks a winner", func() {
			BeforeEach(func() {
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{LRPs: []auctiontypes.LRPAuction{startAuction}})
			})

			It("picks the best cell for the job", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(1))

				startsToB := clients["B"].PerformArgsForCall(0).LRPs

				Ω(startsToB).Should(ConsistOf(startAuction))
			})

			It("marks the start auction as succeeded", func() {
				startAuction.Winner = "B"
				startAuction.Attempts = 1
				startAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulLRPs).Should(ConsistOf(startAuction))
				Ω(results.FailedLRPs).Should(BeEmpty())

				itBumpsTheStartedCounters(1, 0)
			})
		})

		Context("when the cell rejects the start auction", func() {
			BeforeEach(func() {
				clients["B"].PerformReturns(auctiontypes.Work{LRPs: []auctiontypes.LRPAuction{startAuction}}, nil)
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{LRPs: []auctiontypes.LRPAuction{startAuction}})
			})

			It("marks the start auction as failed", func() {
				startAuction.Attempts = 1
				Ω(results.SuccessfulLRPs).Should(BeEmpty())
				Ω(results.FailedLRPs).Should(ConsistOf(startAuction))

				itBumpsTheStartedCounters(0, 0)
			})
		})

		Context("when there is no room", func() {
			BeforeEach(func() {
				startAuction = BuildLRPAuction("pg-4", 0, "lucid64", 1000, 1000, timeProvider.Now())
				timeProvider.Increment(time.Minute)
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{LRPs: []auctiontypes.LRPAuction{startAuction}})
			})

			It("should not attempt to start the LRP", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
			})

			It("should mark the start auction as failed", func() {
				startAuction.Attempts = 1
				Ω(results.SuccessfulLRPs).Should(BeEmpty())
				Ω(results.FailedLRPs).Should(ConsistOf(startAuction))

				itBumpsTheStartedCounters(0, 0)
			})
		})
	})

	Describe("handling task auctions", func() {
		var taskAuction auctiontypes.TaskAuction

		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"does-not-matter", 0, 10, 10},
				{"does-not-matter", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"does-not-matter", 0, 10, 10},
			}))

			taskAuction = BuildTaskAuction(BuildTask("tg-1", "lucid64", 10, 10), timeProvider.Now())
			timeProvider.Increment(time.Minute)
		})

		Context("when it picks a winner", func() {
			BeforeEach(func() {
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{Tasks: []auctiontypes.TaskAuction{taskAuction}})
			})

			It("picks the best cell for the job", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(1))

				tasksToB := clients["B"].PerformArgsForCall(0).Tasks

				Ω(tasksToB).Should(ConsistOf(
					taskAuction.Task,
				))
			})

			It("marks the task auction as succeeded", func() {
				Ω(results.SuccessfulTasks).Should(HaveLen(1))
				successfulTask := results.SuccessfulTasks[0]
				Ω(successfulTask.Winner).Should(Equal("B"))
				Ω(successfulTask.Attempts).Should(Equal(1))
				Ω(successfulTask.WaitDuration).Should(Equal(time.Minute))

				Ω(results.FailedTasks).Should(BeEmpty())

				itBumpsTheStartedCounters(0, 1)
			})
		})

		Context("when the cell rejects the task", func() {
			BeforeEach(func() {
				clients["B"].PerformReturns(auctiontypes.Work{Tasks: []models.Task{taskAuction.Task}}, nil)
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{Tasks: []auctiontypes.TaskAuction{taskAuction}})
			})

			It("marks the task auction as failed", func() {
				Ω(results.SuccessfulTasks).Should(BeEmpty())

				Ω(results.FailedTasks).Should(HaveLen(1))
				failedTask := results.FailedTasks[0]
				Ω(failedTask.Attempts).Should(Equal(1))

				itBumpsTheStartedCounters(0, 0)
			})
		})

		Context("when there is no room", func() {
			BeforeEach(func() {
				taskAuction = BuildTaskAuction(BuildTask("tg-1", "lucid64", 1000, 1000), timeProvider.Now())
				timeProvider.Increment(time.Minute)
				results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{Tasks: []auctiontypes.TaskAuction{taskAuction}})
			})

			It("should not attempt to start the task", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
			})

			It("should mark the start auction as failed", func() {
				Ω(results.SuccessfulTasks).Should(BeEmpty())

				Ω(results.FailedTasks).Should(HaveLen(1))
				failedTask := results.FailedTasks[0]
				Ω(failedTask.Attempts).Should(Equal(1))

				itBumpsTheStartedCounters(0, 0)
			})
		})
	})

	Describe("a comprehensive scenario", func() {
		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", 0, 10, 10},
				{"pg-2", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-3", 0, 10, 10},
				{"pg-4", 0, 20, 20},
			}))
		})

		It("should optimize the distribution", func() {
			startPG3 := BuildLRPAuction(
				"pg-3", 1, "lucid64", 40, 40,
				timeProvider.Now(),
			)
			startPG2 := BuildLRPAuction(
				"pg-2", 1, "lucid64", 5, 5,
				timeProvider.Now(),
			)
			startPGNope := BuildLRPAuction(
				"pg-nope", 1, ".net", 10, 10,
				timeProvider.Now(),
			)

			taskAuction1 := BuildTaskAuction(
				BuildTask("tg-1", "lucid64", 40, 40),
				timeProvider.Now(),
			)
			taskAuction2 := BuildTaskAuction(
				BuildTask("tg-2", "lucid64", 5, 5),
				timeProvider.Now(),
			)
			taskAuctionNope := BuildTaskAuction(
				BuildTask("tg-nope", ".net", 1, 1),
				timeProvider.Now(),
			)

			auctionRequest := auctiontypes.AuctionRequest{
				LRPs:  []auctiontypes.LRPAuction{startPG3, startPG2, startPGNope},
				Tasks: []auctiontypes.TaskAuction{taskAuction1, taskAuction2, taskAuctionNope},
			}

			results = Schedule(workPool, cells, timeProvider, auctionRequest)

			Ω(clients["A"].PerformCallCount()).Should(Equal(1))
			Ω(clients["B"].PerformCallCount()).Should(Equal(1))

			Ω(clients["A"].PerformArgsForCall(0).LRPs).Should(ConsistOf(startPG3))
			Ω(clients["B"].PerformArgsForCall(0).LRPs).Should(ConsistOf(startPG2))

			Ω(clients["A"].PerformArgsForCall(0).Tasks).Should(ConsistOf(taskAuction2.Task))
			Ω(clients["B"].PerformArgsForCall(0).Tasks).Should(ConsistOf(taskAuction1.Task))

			startPG3.Winner = "A"
			startPG3.Attempts = 1
			startPG2.Winner = "B"
			startPG2.Attempts = 1
			Ω(results.SuccessfulLRPs).Should(ConsistOf(startPG3, startPG2))

			Ω(results.SuccessfulTasks).Should(HaveLen(2))
			var successfulTaskAuction1, successfulTaskAuction2 auctiontypes.TaskAuction
			for _, ta := range results.SuccessfulTasks {
				if ta.Identifier() == taskAuction1.Identifier() {
					successfulTaskAuction1 = ta
				} else if ta.Identifier() == taskAuction2.Identifier() {
					successfulTaskAuction2 = ta
				}
			}
			Ω(successfulTaskAuction1).ShouldNot(BeNil())
			Ω(successfulTaskAuction1.Attempts).Should(Equal(1))
			Ω(successfulTaskAuction1.Winner).Should(Equal("B"))
			Ω(successfulTaskAuction2).ShouldNot(BeNil())
			Ω(successfulTaskAuction2.Attempts).Should(Equal(1))
			Ω(successfulTaskAuction2.Winner).Should(Equal("A"))

			startPGNope.Attempts = 1
			Ω(results.FailedLRPs).Should(ConsistOf(startPGNope))
			Ω(results.FailedTasks).Should(HaveLen(1))

			failedTask := results.FailedTasks[0]
			Ω(failedTask.Identifier()).Should(Equal(taskAuctionNope.Identifier()))
			Ω(failedTask.Attempts).Should(Equal(1))

			itBumpsTheStartedCounters(2, 2)
		})
	})

	Describe("ordering work", func() {
		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", 0, 30, 30},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{}))
		})

		It("orders work such that large start auctions occur first", func() {
			startMedium := BuildLRPAuction(
				"pg-medium", 1, "lucid64", 40, 40,
				timeProvider.Now(),
			)
			startLarge := BuildLRPAuction(
				"pg-large", 1, "lucid64", 80, 80,
				timeProvider.Now(),
			)
			lrpStartAuctions := []auctiontypes.LRPAuction{startLarge, startMedium} //note we're submitting the smaller one first

			results = Schedule(workPool, cells, timeProvider, auctiontypes.AuctionRequest{LRPs: lrpStartAuctions})

			Ω(results.FailedLRPs).Should(BeEmpty())

			Ω(clients["A"].PerformCallCount()).Should(Equal(1))
			Ω(clients["B"].PerformCallCount()).Should(Equal(1))

			Ω(clients["A"].PerformArgsForCall(0).LRPs).Should(ConsistOf(startMedium))
			Ω(clients["B"].PerformArgsForCall(0).LRPs).Should(ConsistOf(startLarge))

			startMedium.Winner = "A"
			startMedium.Attempts = 1
			startLarge.Winner = "B"
			startLarge.Attempts = 1
			Ω(results.SuccessfulLRPs).Should(ConsistOf(startMedium, startLarge))

			itBumpsTheStartedCounters(2, 0)
		})
	})
})
