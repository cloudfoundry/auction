package auction_http_handlers_test

import (
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("State", func() {
	Context("when the state call succeeds", func() {
		var repState auctiontypes.RepState
		BeforeEach(func() {
			repState = auctiontypes.RepState{
				Stack: "lucid64",
			}
			auctionRep.StateReturns(repState, nil)
			Ω(auctionRep.StateCallCount()).Should(Equal(0))
		})

		It("it returns whatever the state call returns", func() {
			status, body := Request(routes.State, nil, nil)
			Ω(status).Should(Equal(http.StatusOK))

			Ω(body).Should(MatchJSON(JSONFor(repState)))

			Ω(auctionRep.StateCallCount()).Should(Equal(1))
		})
	})

	Context("when the state call fails", func() {
		It("fails", func() {
			auctionRep.StateReturns(auctiontypes.RepState{}, errors.New("boom"))
			Ω(auctionRep.StateCallCount()).Should(Equal(0))

			status, body := Request(routes.State, nil, nil)
			Ω(status).Should(Equal(http.StatusInternalServerError))
			Ω(body).Should(BeEmpty())

			Ω(auctionRep.StateCallCount()).Should(Equal(1))
		})
	})
})
