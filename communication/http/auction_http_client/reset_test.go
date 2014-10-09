package auction_http_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) Reset", func() {
	It("should tell the rep to reset", func() {
		client.Reset(RepAddressFor("A"))
		Ω(auctionRepA.ResetCallCount()).Should(Equal(1))
	})

	Context("when a request doesn't succeed", func() {
		It("does not explode", func() {
			client.Reset(RepAddressFor("RepThat500s"))
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("does not explode", func() {
			client.Reset(RepAddressFor("RepThatErrors"))
		})
	})
})
