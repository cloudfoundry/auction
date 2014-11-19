package auction_http_handlers_test

import (
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reset Handler", func() {
	Describe("Reset", func() {
		Context("when the reset succeeds", func() {
			It("succeeds", func() {
				Ω(auctionRep.ResetCallCount()).Should(Equal(0))

				status, body := Request(routes.Sim_Reset, nil, nil)
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(BeEmpty())

				Ω(auctionRep.ResetCallCount()).Should(Equal(1))
			})
		})

		Context("when the reset fails", func() {
			It("fails", func() {
				auctionRep.ResetReturns(errors.New("boom"))
				Ω(auctionRep.ResetCallCount()).Should(Equal(0))

				status, body := Request(routes.Sim_Reset, nil, nil)
				Ω(status).Should(Equal(http.StatusInternalServerError))
				Ω(body).Should(BeEmpty())

				Ω(auctionRep.ResetCallCount()).Should(Equal(1))
			})
		})
	})
})
