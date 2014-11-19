package auctionrunner_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch", func() {
	var startAuction models.LRPStartAuction
	var stopAuction models.LRPStopAuction
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
	})

	It("should start off empty", func() {
		Ω(batch.HasWork).ShouldNot(Receive())
		starts, stops := batch.DedupeAndDrain()
		Ω(starts).Should(BeEmpty())
		Ω(stops).Should(BeEmpty())
	})

	Describe("adding work", func() {
		Context("when adding start auctions", func() {
			BeforeEach(func() {
				startAuction = BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10)
				batch.AddLRPStartAuction(startAuction)
			})

			It("makes the start auction available when drained", func() {
				startAuctions, _ := batch.DedupeAndDrain()
				Ω(startAuctions).Should(ConsistOf(BuildStartAuction(startAuction, timeProvider.Time())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("when adding stop auctions", func() {
			BeforeEach(func() {
				stopAuction = BuildLRPStopAuction("pg-1", 1)
				batch.AddLRPStopAuction(stopAuction)
			})

			It("makes the stop auction available when drained", func() {
				_, stopAuctions := batch.DedupeAndDrain()
				Ω(stopAuctions).Should(ConsistOf(BuildStopAuction(stopAuction, timeProvider.Time())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})
	})

	Describe("DedupeAndDrain", func() {
		BeforeEach(func() {
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-2", "ig-2", 2, "lucid64", 10, 10))

			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-1", 1))
			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-1", 1))
			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-2", 3))
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			startAuctions, stopAuctions := batch.DedupeAndDrain()
			Ω(startAuctions).Should(ConsistOf(
				BuildStartAuction(
					BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10),
					timeProvider.Time(),
				),
				BuildStartAuction(
					BuildLRPStartAuction("pg-2", "ig-2", 2, "lucid64", 10, 10),
					timeProvider.Time(),
				),
			))

			Ω(stopAuctions).Should(ConsistOf(
				BuildStopAuction(
					BuildLRPStopAuction("pg-1", 1),
					timeProvider.Time(),
				),
				BuildStopAuction(
					BuildLRPStopAuction("pg-2", 3),
					timeProvider.Time(),
				),
			))
		})

		It("should clear out its cache, so a subsequent call shouldn't fetch anything", func() {
			batch.DedupeAndDrain()
			startAuctions, stopAuctions := batch.DedupeAndDrain()
			Ω(startAuctions).Should(BeEmpty())
			Ω(stopAuctions).Should(BeEmpty())
		})
	})
})
