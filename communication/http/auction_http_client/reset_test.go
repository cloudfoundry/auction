package auction_http_client_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) Reset", func() {
	It("should tell the rep to reset", func() {
		client.Reset()
		Expect(auctionRep.ResetCallCount()).To(Equal(1))
	})

	Context("when the request succeeds", func() {
		BeforeEach(func() {
			auctionRep.ResetReturns(nil)
		})

		It("should return the state returned by the rep", func() {
			Expect(client.Reset()).To(Succeed())
		})
	})

	Context("when the request fails", func() {
		BeforeEach(func() {
			auctionRep.ResetReturns(errors.New("boom"))
		})

		It("should error", func() {
			Expect(client.Reset()).NotTo(Succeed())
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("should error", func() {
			Expect(clientForServerThatErrors.Reset()).NotTo(Succeed())
		})
	})
})
