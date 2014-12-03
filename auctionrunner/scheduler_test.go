package auctionrunner_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/gunk/workpool"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scheudler", func() {
	var clients map[string]*fakes.FakeSimulationCellRep
	var cells map[string]*Cell
	var timeProvider *faketimeprovider.FakeTimeProvider
	var workPool *workpool.WorkPool
	var results auctiontypes.AuctionResults

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		workPool = workpool.NewWorkPool(5)

		clients = map[string]*fakes.FakeSimulationCellRep{}
		cells = map[string]*Cell{}
	})

	AfterEach(func() {
		workPool.Stop()
	})

	Context("when the cells are empty", func() {
		It("immediately returns everything as having failed, incrementing the attempt number", func() {
			startAuction := BuildStartAuction(
				BuildLRPStartAuction("pg-7", "ig-7", 0, "lucid64", 10, 10),
				timeProvider.Now(),
			)

			startAuctions := []auctiontypes.StartAuction{startAuction}

			stopAuction := BuildStopAuction(
				BuildLRPStopAuction("pg-1", 1),
				timeProvider.Now(),
			)

			stopAuctions := []auctiontypes.StopAuction{stopAuction}

			results := Schedule(workPool, map[string]*Cell{}, timeProvider, startAuctions, stopAuctions)
			Ω(results.SuccessfulStarts).Should(BeEmpty())
			Ω(results.SuccessfulStops).Should(BeEmpty())
			startAuction.Attempts = 1
			stopAuction.Attempts = 1
			Ω(results.FailedStarts).Should(ConsistOf(startAuction))
			Ω(results.FailedStops).Should(ConsistOf(stopAuction))
		})
	})

	Describe("handling start auctions", func() {
		var startAuction auctiontypes.StartAuction

		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", "ig-1", 0, 10, 10},
				{"pg-2", "ig-2", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-3", "ig-3", 0, 10, 10},
			}))

			startAuction = BuildStartAuction(BuildLRPStartAuction("pg-4", "ig-4", 0, "lucid64", 10, 10), timeProvider.Now())
			timeProvider.Increment(time.Minute)
		})

		Context("when it picks a winner", func() {
			BeforeEach(func() {
				results = Schedule(workPool, cells, timeProvider, []auctiontypes.StartAuction{startAuction}, nil)
			})

			It("picks the best cell for the job", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(1))

				startsToB := clients["B"].PerformArgsForCall(0).Starts

				Ω(startsToB).Should(ConsistOf(
					startAuction.LRPStartAuction,
				))
			})

			It("marks the start auction as succeeded", func() {
				startAuction.Winner = "B"
				startAuction.Attempts = 1
				startAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulStarts).Should(ConsistOf(startAuction))
				Ω(results.FailedStarts).Should(BeEmpty())
			})
		})

		Context("when the cell rejects the start auction", func() {
			BeforeEach(func() {
				clients["B"].PerformReturns(auctiontypes.Work{Starts: []models.LRPStartAuction{startAuction.LRPStartAuction}}, nil)
				results = Schedule(workPool, cells, timeProvider, []auctiontypes.StartAuction{startAuction}, nil)
			})

			It("marks the start auction as failed", func() {
				startAuction.Attempts = 1
				Ω(results.SuccessfulStarts).Should(BeEmpty())
				Ω(results.FailedStarts).Should(ConsistOf(startAuction))
			})
		})

		Context("when there is no room", func() {
			BeforeEach(func() {
				startAuction = BuildStartAuction(BuildLRPStartAuction("pg-4", "ig-4", 0, "lucid64", 1000, 1000), timeProvider.Now())
				timeProvider.Increment(time.Minute)
				results = Schedule(workPool, cells, timeProvider, []auctiontypes.StartAuction{startAuction}, nil)
			})

			It("should not attempt to start the LRP", func() {
				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
			})

			It("should mark the start auction as failed", func() {
				startAuction.Attempts = 1
				Ω(results.SuccessfulStarts).Should(BeEmpty())
				Ω(results.FailedStarts).Should(ConsistOf(startAuction))
			})
		})
	})

	Describe("handling stop auctions", func() {
		var stopAuction auctiontypes.StopAuction

		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg", "ig-1", 0, 10, 10},
				{"pg", "ig-2", 1, 10, 10},
				{"pg", "ig-3", 1, 10, 10},
				{"pg-one", "ig-4", 0, 10, 10},
				{"pg-other", "ig-5", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg", "ig-6", 1, 10, 10},
				{"pg-other", "ig-7", 0, 10, 10},
			}))

			clients["C"] = &fakes.FakeSimulationCellRep{}
			cells["C"] = NewCell(clients["C"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg", "ig-8", 1, 10, 10},
				{"pg-other", "ig-9", 0, 10, 10},
				{"pg-other", "ig-10", 0, 10, 10},
				{"pg-three", "ig-11", 2, 10, 10},
				{"pg-three", "ig-12", 2, 10, 10},
				{"pg-three", "ig-12", 2, 10, 10},
			}))
		})

		Context("when the stop auction maps onto multiple instances", func() {
			BeforeEach(func() {
				stopAuction = BuildStopAuction(
					BuildLRPStopAuction("pg", 1),
					timeProvider.Now(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("tells the appropriate cells to stop", func() {
				results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				Ω(clients["A"].PerformCallCount()).Should(Equal(1))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
				Ω(clients["C"].PerformCallCount()).Should(Equal(1))

				stopsToA := clients["A"].PerformArgsForCall(0).Stops
				stopsToC := clients["C"].PerformArgsForCall(0).Stops

				Ω(stopsToA).Should(ConsistOf(
					models.StopLRPInstance{
						ProcessGuid:  "pg",
						InstanceGuid: "ig-2",
						Index:        1,
					},
					models.StopLRPInstance{
						ProcessGuid:  "pg",
						InstanceGuid: "ig-3",
						Index:        1,
					},
				))

				Ω(stopsToC).Should(ConsistOf(models.StopLRPInstance{
					ProcessGuid:  "pg",
					InstanceGuid: "ig-8",
					Index:        1,
				}))
			})

			It("marks the stop auction a success", func() {
				results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				stopAuction.Winner = "B"
				stopAuction.Attempts = 1
				stopAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
				Ω(results.FailedStops).Should(BeEmpty())
			})

			Context("if a cell fails to stop", func() {
				It("nonetheless markes the stop auction as a success -- if this is really an issue it will come up again later", func() {
					clients["C"].PerformReturns(auctiontypes.Work{}, errors.New("boom"))
					results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

					stopAuction.Winner = "B"
					stopAuction.Attempts = 1
					stopAuction.WaitDuration = time.Minute
					Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
					Ω(results.FailedStops).Should(BeEmpty())
				})
			})
		})

		Context("when the stop auction maps onto a single cell with multiple instances", func() {
			BeforeEach(func() {
				stopAuction = BuildStopAuction(
					BuildLRPStopAuction("pg-three", 2),
					timeProvider.Now(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("stops all but one of the instances (doesn't matter which)", func() {
				results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
				Ω(clients["C"].PerformCallCount()).Should(Equal(1))

				stopsToC := clients["C"].PerformArgsForCall(0).Stops

				Ω(stopsToC).Should(HaveLen(2))

				stopAuction.Winner = "C"
				stopAuction.Attempts = 1
				stopAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
				Ω(results.FailedStops).Should(BeEmpty())
			})
		})

		Context("when the stop auction maps onto a single instance", func() {
			BeforeEach(func() {
				stopAuction = BuildStopAuction(
					BuildLRPStopAuction("pg-one", 0),
					timeProvider.Now(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("succeeds without taking any actions on any cells", func() {
				results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
				Ω(clients["C"].PerformCallCount()).Should(Equal(0))

				stopAuction.Winner = "A"
				stopAuction.Attempts = 1
				stopAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
				Ω(results.FailedStops).Should(BeEmpty())
			})
		})

		Context("when no instances are found for the stop auction", func() {
			BeforeEach(func() {
				stopAuction = BuildStopAuction(
					BuildLRPStopAuction("pg", 17),
					timeProvider.Now(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("fails silently -- if this is really an issue it will come up again later", func() {
				results = Schedule(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				Ω(clients["A"].PerformCallCount()).Should(Equal(0))
				Ω(clients["B"].PerformCallCount()).Should(Equal(0))
				Ω(clients["C"].PerformCallCount()).Should(Equal(0))

				stopAuction.Attempts = 1
				stopAuction.WaitDuration = time.Minute

				Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
				Ω(results.FailedStops).Should(BeEmpty())
			})
		})
	})

	Describe("a comprehensive scenario", func() {
		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", "ig-1", 0, 10, 10},
				{"pg-2", "ig-2", 0, 10, 10},
				{"pg-dupe", "ig-3", 0, 80, 80},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-3", "ig-4", 0, 10, 10},
				{"pg-dupe", "ig-5", 0, 80, 80},
			}))
		})

		It("should optimize the distribution", func() {
			stopAuctions := []auctiontypes.StopAuction{
				BuildStopAuction(
					BuildLRPStopAuction("pg-dupe", 0),
					timeProvider.Now(),
				),
			}

			startPG3 := BuildStartAuction(
				BuildLRPStartAuction("pg-3", "ig-new-1", 1, "lucid64", 40, 40),
				timeProvider.Now(),
			)
			startPG2 := BuildStartAuction(
				BuildLRPStartAuction("pg-2", "ig-new-2", 1, "lucid64", 10, 10),
				timeProvider.Now(),
			)
			startPGNope := BuildStartAuction(
				BuildLRPStartAuction("pg-nope", "ig-nope", 1, ".net", 10, 10),
				timeProvider.Now(),
			)
			startAuctions := []auctiontypes.StartAuction{startPG3, startPG2, startPGNope}

			results = Schedule(workPool, cells, timeProvider, startAuctions, stopAuctions)

			Ω(clients["A"].PerformCallCount()).Should(Equal(1))
			Ω(clients["B"].PerformCallCount()).Should(Equal(1))

			Ω(clients["A"].PerformArgsForCall(0).Stops).Should(ConsistOf(models.StopLRPInstance{
				ProcessGuid:  "pg-dupe",
				InstanceGuid: "ig-3",
				Index:        0,
			}))
			Ω(clients["B"].PerformArgsForCall(0).Stops).Should(BeEmpty())

			Ω(clients["A"].PerformArgsForCall(0).Starts).Should(ConsistOf(startPG3.LRPStartAuction))
			Ω(clients["B"].PerformArgsForCall(0).Starts).Should(ConsistOf(startPG2.LRPStartAuction))

			successfulStop := stopAuctions[0]
			successfulStop.Winner = "B"
			successfulStop.Attempts = 1
			Ω(results.SuccessfulStops).Should(ConsistOf(successfulStop))

			startPG3.Winner = "A"
			startPG3.Attempts = 1
			startPG2.Winner = "B"
			startPG2.Attempts = 1
			Ω(results.SuccessfulStarts).Should(ConsistOf(startPG3, startPG2))

			Ω(results.FailedStops).Should(BeEmpty())
			startPGNope.Attempts = 1
			Ω(results.FailedStarts).Should(ConsistOf(startPGNope))
		})
	})

	Describe("ordering work", func() {
		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationCellRep{}
			cells["A"] = NewCell(clients["A"], BuildCellState(100, 100, 100, []auctiontypes.LRP{
				{"pg-1", "ig-1", 0, 30, 30},
			}))

			clients["B"] = &fakes.FakeSimulationCellRep{}
			cells["B"] = NewCell(clients["B"], BuildCellState(100, 100, 100, []auctiontypes.LRP{}))
		})

		It("orders work such that large start auctions occur first", func() {
			startMedium := BuildStartAuction(
				BuildLRPStartAuction("pg-medium", "ig-medium", 1, "lucid64", 40, 40),
				timeProvider.Now(),
			)
			startLarge := BuildStartAuction(
				BuildLRPStartAuction("pg-large", "ig-large", 1, "lucid64", 80, 80),
				timeProvider.Now(),
			)
			startAuctions := []auctiontypes.StartAuction{startLarge, startMedium} //note we're submitting the smaller one first

			results = Schedule(workPool, cells, timeProvider, startAuctions, nil)

			Ω(results.FailedStarts).Should(BeEmpty())

			startMedium.Winner = "A"
			startMedium.Attempts = 1
			startLarge.Winner = "B"
			startLarge.Attempts = 1
			Ω(results.SuccessfulStarts).Should(ConsistOf(startMedium, startLarge))

			Ω(clients["A"].PerformCallCount()).Should(Equal(1))
			Ω(clients["B"].PerformCallCount()).Should(Equal(1))

			Ω(clients["A"].PerformArgsForCall(0).Starts).Should(ConsistOf(startMedium.LRPStartAuction))
			Ω(clients["B"].PerformArgsForCall(0).Starts).Should(ConsistOf(startLarge.LRPStartAuction))
		})
	})
})
