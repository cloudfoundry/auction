package auctionrunner_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FailedWorkResubmitter", func() {
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider
	var results auctiontypes.AuctionResults
	var maxRetries int

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
		maxRetries = 3
	})

	It("always returns succesful work untouched", func() {
		results = auctiontypes.AuctionResults{
			SuccessfulStarts: []auctiontypes.StartAuction{
				BuildStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10), timeProvider.Time()),
				BuildStartAuction(BuildLRPStartAuction("pg-2", "ig-2", 1, "lucid64", 10, 10), timeProvider.Time()),
			},
			SuccessfulStops: []auctiontypes.StopAuction{
				BuildStopAuction(BuildLRPStopAuction("pg-1", 2), timeProvider.Time()),
				BuildStopAuction(BuildLRPStopAuction("pg-2", 2), timeProvider.Time()),
			},
			FailedStarts: []auctiontypes.StartAuction{},
			FailedStops:  []auctiontypes.StopAuction{},
		}

		out := ResubmitFailedWork(batch, results, maxRetries)
		Ω(out).Should(Equal(results))
	})

	Context("if there is failed work", func() {
		var retryableStartAuction, failedStartAuction auctiontypes.StartAuction
		var retryableStopAuction, failedStopAuction auctiontypes.StopAuction

		BeforeEach(func() {
			retryableStartAuction = BuildStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10), timeProvider.Time())
			retryableStartAuction.Attempts = maxRetries
			failedStartAuction = BuildStartAuction(BuildLRPStartAuction("pg-2", "ig-2", 1, "lucid64", 10, 10), timeProvider.Time())
			failedStartAuction.Attempts = maxRetries + 1

			retryableStopAuction = BuildStopAuction(BuildLRPStopAuction("pg-1", 2), timeProvider.Time())
			retryableStopAuction.Attempts = maxRetries
			failedStopAuction = BuildStopAuction(BuildLRPStopAuction("pg-2", 2), timeProvider.Time())
			failedStopAuction.Attempts = maxRetries + 1

			results = auctiontypes.AuctionResults{
				FailedStarts: []auctiontypes.StartAuction{retryableStartAuction, failedStartAuction},
				FailedStops:  []auctiontypes.StopAuction{retryableStopAuction, failedStopAuction},
			}
		})

		It("should resubmit work that can be retried and does not return it, but returns work that has exceeded maxretries without resubmitting it", func() {
			out := ResubmitFailedWork(batch, results, maxRetries)
			Ω(out.FailedStarts).Should(ConsistOf(failedStartAuction))
			Ω(out.FailedStops).Should(ConsistOf(failedStopAuction))

			resubmittedStarts, resubmittedStops := batch.DedupeAndDrain()
			Ω(resubmittedStarts).Should(ConsistOf(retryableStartAuction))
			Ω(resubmittedStops).Should(ConsistOf(retryableStopAuction))
		})
	})
})
