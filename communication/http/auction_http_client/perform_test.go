package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Perform", func() {
	var work, failedWork auctiontypes.Work

	BeforeEach(func() {
		work = auctiontypes.Work{
			Stops: []models.StopLRPInstance{
				{
					ProcessGuid:  "pg-a",
					InstanceGuid: "ig-a",
					Index:        1,
				},
				{
					ProcessGuid:  "pg-b",
					InstanceGuid: "ig-b",
					Index:        2,
				},
			},
		}

		failedWork = auctiontypes.Work{
			Stops: []models.StopLRPInstance{
				{
					ProcessGuid:  "pg-a",
					InstanceGuid: "ig-a",
					Index:        1,
				},
			},
		}
	})

	It("should tell the rep to perform", func() {
		Ω(auctionRep.PerformCallCount()).Should(Equal(0))
		client.Perform(work)
		Ω(auctionRep.PerformArgsForCall(0)).Should(Equal(work))
	})

	Context("when the request succeeds", func() {
		BeforeEach(func() {
			auctionRep.PerformReturns(failedWork, nil)
		})

		It("should return the state returned by the rep", func() {
			Ω(client.Perform(work)).Should(Equal(failedWork))
		})
	})

	Context("when the request fails", func() {
		BeforeEach(func() {
			auctionRep.PerformReturns(failedWork, errors.New("boom"))
		})

		It("should error", func() {
			failedWork, err := client.Perform(work)
			Ω(failedWork).Should(BeZero())
			Ω(err).Should(HaveOccurred())
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("should error", func() {
			failedWork, err := clientForServerThatErrors.Perform(work)
			Ω(failedWork).Should(BeZero())
			Ω(err).Should(HaveOccurred())
		})
	})
})
