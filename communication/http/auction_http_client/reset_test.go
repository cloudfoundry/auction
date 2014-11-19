package auction_http_client_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) Reset", func() {
	It("should tell the rep to reset", func() {
		client.Reset()
		Ω(auctionRep.ResetCallCount()).Should(Equal(1))
	})

	Context("when the request succeeds", func() {
		BeforeEach(func() {
			auctionRep.ResetReturns(nil)
		})

		It("should return the state returned by the rep", func() {
			Ω(client.Reset()).Should(Succeed())
		})
	})

	Context("when the request fails", func() {
		BeforeEach(func() {
			auctionRep.ResetReturns(errors.New("boom"))
		})

		It("should error", func() {
			Ω(client.Reset()).ShouldNot(Succeed())
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("should error", func() {
			Ω(clientForServerThatErrors.Reset()).ShouldNot(Succeed())
		})
	})
})
