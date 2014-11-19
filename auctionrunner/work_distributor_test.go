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

var _ = Describe("WorkDistributor", func() {
	var clients map[string]*fakes.FakeSimulationAuctionRep
	var cells map[string]*Cell
	var timeProvider *faketimeprovider.FakeTimeProvider
	var workPool *workpool.WorkPool
	var results DistributeWorkResults

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		workPool = workpool.NewWorkPool(5)

		clients = map[string]*fakes.FakeSimulationAuctionRep{}
		cells = map[string]*Cell{}
	})

	Context("when the cells are empty", func() {
		It("immediately returns everything as having failed, incrementing the attempt number", func() {
			startAuction := BuildStartAuction(
				BuildLRPStartAuction("pg-7", "ig-7", 0, "lucid64", 10, 10),
				timeProvider.Time(),
			)

			startAuctions := []auctiontypes.StartAuction{startAuction}

			stopAuction := BuildStopAuction(
				BuildLRPStopAuction("pg-1", 1),
				timeProvider.Time(),
			)

			stopAuctions := []auctiontypes.StopAuction{stopAuction}

			results := DistributeWork(workPool, map[string]*Cell{}, timeProvider, startAuctions, stopAuctions)
			Ω(results.SuccessfulStarts).Should(BeEmpty())
			Ω(results.SuccessfulStops).Should(BeEmpty())
			startAuction.Attempts = 1
			stopAuction.Attempts = 1
			Ω(results.FailedStarts).Should(ConsistOf(startAuction))
			Ω(results.FailedStops).Should(ConsistOf(stopAuction))
		})
	})

	Describe("handling stop auctions", func() {
		var stopAuction auctiontypes.StopAuction

		BeforeEach(func() {
			clients["A"] = &fakes.FakeSimulationAuctionRep{}
			cells["A"] = NewCell(clients["A"], BuildRepState(100, 100, 100, []auctiontypes.LRP{
				{"pg", "ig-1", 0, 10, 10},
				{"pg", "ig-2", 1, 10, 10},
				{"pg", "ig-3", 1, 10, 10},
				{"pg-one", "ig-4", 0, 10, 10},
				{"pg-other", "ig-5", 0, 10, 10},
			}))

			clients["B"] = &fakes.FakeSimulationAuctionRep{}
			cells["B"] = NewCell(clients["B"], BuildRepState(100, 100, 100, []auctiontypes.LRP{
				{"pg", "ig-6", 1, 10, 10},
				{"pg-other", "ig-7", 0, 10, 10},
			}))

			clients["C"] = &fakes.FakeSimulationAuctionRep{}
			cells["C"] = NewCell(clients["C"], BuildRepState(100, 100, 100, []auctiontypes.LRP{
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
					timeProvider.Time(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("tells the appropriate cells to stop", func() {
				results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

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
				results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

				stopAuction.Winner = "B"
				stopAuction.Attempts = 1
				stopAuction.WaitDuration = time.Minute
				Ω(results.SuccessfulStops).Should(ConsistOf(stopAuction))
				Ω(results.FailedStops).Should(BeEmpty())
			})

			Context("if a cell fails to stop", func() {
				It("nonetheless markes the stop auction as a success -- if this is really an issue it will come up again later", func() {
					clients["C"].PerformReturns(auctiontypes.Work{}, errors.New("boom"))
					results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

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
					timeProvider.Time(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("stops all but one of the instances (doesn't matter which)", func() {
				results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

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
					timeProvider.Time(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("succeeds without taking any actions on any cells", func() {
				results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

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
					timeProvider.Time(),
				)
				timeProvider.Increment(time.Minute)
			})

			It("fails silently -- if this is really an issue it will come up again later", func() {
				results = DistributeWork(workPool, cells, timeProvider, nil, []auctiontypes.StopAuction{stopAuction})

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
})
