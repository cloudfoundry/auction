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
				Expect(auctionRep.ResetCallCount()).To(Equal(0))

				status, body := Request(routes.Sim_Reset, nil, nil)
				Expect(status).To(Equal(http.StatusOK))
				Expect(body).To(BeEmpty())

				Expect(auctionRep.ResetCallCount()).To(Equal(1))
			})
		})

		Context("when the reset fails", func() {
			It("fails", func() {
				auctionRep.ResetReturns(errors.New("boom"))
				Expect(auctionRep.ResetCallCount()).To(Equal(0))

				status, body := Request(routes.Sim_Reset, nil, nil)
				Expect(status).To(Equal(http.StatusInternalServerError))
				Expect(body).To(BeEmpty())

				Expect(auctionRep.ResetCallCount()).To(Equal(1))
			})
		})
	})
})
