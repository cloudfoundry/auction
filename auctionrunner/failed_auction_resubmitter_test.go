package auctionrunner_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResubmitFailedAuctions", func() {
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider
	var results auctiontypes.AuctionResults
	var maxRetries int
	var metricSender *fake.FakeMetricSender

	BeforeEach(func() {
		metricSender = fake.NewFakeMetricSender()
		metrics.Initialize(metricSender)

		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
		maxRetries = 3
	})

	It("always returns succesful work untouched", func() {
		results = auctiontypes.AuctionResults{
			SuccessfulLRPs: []auctiontypes.LRPAuction{
				BuildLRPAuction("pg-1", 1, "lucid64", 10, 10, timeProvider.Now()),
				BuildLRPAuction("pg-2", 1, "lucid64", 10, 10, timeProvider.Now()),
			},
			SuccessfulTasks: []auctiontypes.TaskAuction{
				BuildTaskAuction(BuildTask("tg-1", "lucid64", 10, 10), timeProvider.Now()),
				BuildTaskAuction(BuildTask("tg-2", "lucid64", 10, 10), timeProvider.Now()),
			},
			FailedLRPs:  []auctiontypes.LRPAuction{},
			FailedTasks: []auctiontypes.TaskAuction{},
		}

		out := ResubmitFailedAuctions(batch, results, maxRetries)
		Ω(out).Should(Equal(results))
	})

	It("should not resubmit if there is nothing to resubmit", func() {
		ResubmitFailedAuctions(batch, auctiontypes.AuctionResults{}, maxRetries)
		Ω(batch.HasWork).ShouldNot(Receive())
	})

	Context("if there is failed work", func() {
		var retryableStartAuction, failedStartAuction auctiontypes.LRPAuction
		var retryableTaskAuction, failedTaskAuction auctiontypes.TaskAuction

		BeforeEach(func() {
			retryableStartAuction = BuildLRPAuction("pg-1", 1, "lucid64", 10, 10, timeProvider.Now())
			retryableStartAuction.Attempts = maxRetries
			failedStartAuction = BuildLRPAuction("pg-2", 1, "lucid64", 10, 10, timeProvider.Now())
			failedStartAuction.Attempts = maxRetries + 1

			retryableTaskAuction = BuildTaskAuction(BuildTask("tg-1", "lucid64", 10, 10), timeProvider.Now())
			retryableTaskAuction.Attempts = maxRetries
			failedTaskAuction = BuildTaskAuction(BuildTask("tg-2", "lucid64", 10, 10), timeProvider.Now())
			failedTaskAuction.Attempts = maxRetries + 1

			results = auctiontypes.AuctionResults{
				FailedLRPs:  []auctiontypes.LRPAuction{retryableStartAuction, failedStartAuction},
				FailedTasks: []auctiontypes.TaskAuction{retryableTaskAuction, failedTaskAuction},
			}
		})

		It("should resubmit work that can be retried and does not return it, but returns work that has exceeded maxretries without resubmitting it", func() {
			out := ResubmitFailedAuctions(batch, results, maxRetries)
			Ω(out.FailedLRPs).Should(ConsistOf(failedStartAuction))
			Ω(out.FailedTasks).Should(ConsistOf(failedTaskAuction))

			resubmittedStarts, resubmittedTasks := batch.DedupeAndDrain()
			Ω(resubmittedStarts).Should(ConsistOf(retryableStartAuction))
			Ω(resubmittedTasks).Should(ConsistOf(retryableTaskAuction))
		})

		It("should increment fail metrics for the failed auctions", func() {
			ResubmitFailedAuctions(batch, results, maxRetries)

			Ω(metricSender.GetCounter("AuctioneerLRPAuctionsFailed")).Should(BeNumerically("==", 1))
			Ω(metricSender.GetCounter("AuctioneerTaskAuctionsFailed")).Should(BeNumerically("==", 1))
		})
	})
})
