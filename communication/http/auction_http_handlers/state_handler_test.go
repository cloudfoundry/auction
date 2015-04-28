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
		var repState auctiontypes.CellState
		BeforeEach(func() {
			repState = auctiontypes.CellState{
				RootFSProviders: auctiontypes.RootFSProviders{"docker": auctiontypes.ArbitraryRootFSProvider{}},
			}
			auctionRep.StateReturns(repState, nil)
			Expect(auctionRep.StateCallCount()).To(Equal(0))
		})

		It("it returns whatever the state call returns", func() {
			status, body := Request(routes.State, nil, nil)
			Expect(status).To(Equal(http.StatusOK))

			Expect(body).To(MatchJSON(JSONFor(repState)))

			Expect(auctionRep.StateCallCount()).To(Equal(1))
		})
	})

	Context("when the state call fails", func() {
		It("fails", func() {
			auctionRep.StateReturns(auctiontypes.CellState{}, errors.New("boom"))
			Expect(auctionRep.StateCallCount()).To(Equal(0))

			status, body := Request(routes.State, nil, nil)
			Expect(status).To(Equal(http.StatusInternalServerError))
			Expect(body).To(BeEmpty())

			Expect(auctionRep.StateCallCount()).To(Equal(1))
		})
	})
})
