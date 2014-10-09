package auction_http_handlers_test

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BidForStopAuctionHandler", func() {
	Context("with valid JSON", func() {
		var stopAuctionInfo auctiontypes.StopAuctionInfo

		BeforeEach(func() {
			stopAuctionInfo = auctiontypes.StopAuctionInfo{
				ProcessGuid: "process-guid",
				Index:       1,
			}
		})

		It("should notify the auction rep", func() {
			Request(routes.BidForStopAuction, nil, JSONReaderFor(stopAuctionInfo))
			Ω(auctionRep.BidForStopAuctionCallCount()).Should(Equal(1))
			Ω(auctionRep.BidForStopAuctionArgsForCall(0)).Should(Equal(stopAuctionInfo))
		})

		Context("and a succesful bid", func() {
			BeforeEach(func() {
				auctionRep.BidForStopAuctionReturns(0.73, []string{"instance-guid-1", "instance-guid-2"}, nil)
			})

			It("should return the response of the fake", func() {
				status, body := Request(routes.BidForStopAuction, nil, JSONReaderFor(stopAuctionInfo))
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(MatchJSON(JSONFor(auctiontypes.StopAuctionBid{
					Rep:           repGuid,
					Bid:           0.73,
					InstanceGuids: []string{"instance-guid-1", "instance-guid-2"},
					Error:         "",
				})))
			})
		})

		Context("and an unsuccesful bid", func() {
			BeforeEach(func() {
				auctionRep.BidForStopAuctionReturns(0, nil, errors.New("oops"))
			})

			It("should return a non-happy status code and the error", func() {
				status, body := Request(routes.BidForStopAuction, nil, JSONReaderFor(stopAuctionInfo))
				Ω(status).Should(Equal(http.StatusForbidden))
				Ω(body).Should(MatchJSON(JSONFor(auctiontypes.StopAuctionBid{
					Rep:           repGuid,
					Bid:           0,
					InstanceGuids: nil,
					Error:         "oops",
				})))
			})
		})
	})

	Context("when invalid JSON", func() {
		It("should return an error without calling the rep", func() {
			status, body := Request(routes.BidForStopAuction, nil, bytes.NewBufferString("∆"))
			Ω(status).Should(Equal(http.StatusBadRequest))
			Ω(body).Should(ContainSubstring("invalid json: invalid character"))

			Ω(auctionRep.BidForStopAuctionCallCount()).Should(BeZero())
		})
	})
})
