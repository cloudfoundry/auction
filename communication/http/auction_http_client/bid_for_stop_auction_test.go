package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BidForStopAuction", func() {
	var stopAuctionInfo auctiontypes.StopAuctionInfo
	var stopAuctionBids auctiontypes.StopAuctionBids

	BeforeEach(func() {
		stopAuctionInfo = auctiontypes.StopAuctionInfo{
			ProcessGuid: "process-guid",
			Index:       1,
		}

		auctionRepA.BidForStopAuctionReturns(0.27, []string{"A-1", "A-2"}, nil)
	})

	It("should request bids from all passed in reps", func() {
		client.BidForStopAuction(RepAddressesFor("A", "B"), stopAuctionInfo)

		Ω(auctionRepA.BidForStopAuctionCallCount()).Should(Equal(1))
		Ω(auctionRepA.BidForStopAuctionArgsForCall(0)).Should(Equal(stopAuctionInfo))

		Ω(auctionRepB.BidForStopAuctionCallCount()).Should(Equal(1))
		Ω(auctionRepB.BidForStopAuctionArgsForCall(0)).Should(Equal(stopAuctionInfo))
	})

	Context("when the bids are succesful", func() {
		BeforeEach(func() {
			auctionRepB.BidForStopAuctionReturns(0.48, []string{"B-1", "B-2"}, nil)
			stopAuctionBids = client.BidForStopAuction(RepAddressesFor("A", "B"), stopAuctionInfo)
		})

		It("returns all bids", func() {
			Ω(stopAuctionBids).Should(ConsistOf(auctiontypes.StopAuctionBids{
				{Rep: "A", InstanceGuids: []string{"A-1", "A-2"}, Bid: 0.27, Error: ""},
				{Rep: "B", InstanceGuids: []string{"B-1", "B-2"}, Bid: 0.48, Error: ""},
			}))
		})
	})

	Context("when bids are unsuccesful", func() {
		BeforeEach(func() {
			auctionRepB.BidForStopAuctionReturns(0, nil, errors.New("oops"))
			stopAuctionBids = client.BidForStopAuction(RepAddressesFor("A", "B"), stopAuctionInfo)
		})

		It("does not return them", func() {
			Ω(stopAuctionBids).Should(ConsistOf(auctiontypes.StopAuctionBids{
				{Rep: "A", InstanceGuids: []string{"A-1", "A-2"}, Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request doesn't succeed", func() {
		BeforeEach(func() {
			stopAuctionBids = client.BidForStopAuction(RepAddressesFor("A", "RepThat500s"), stopAuctionInfo)
		})

		It("does not return bids that didn't succeed", func() {
			Ω(stopAuctionBids).Should(ConsistOf(auctiontypes.StopAuctionBids{
				{Rep: "A", InstanceGuids: []string{"A-1", "A-2"}, Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request errors (in the network sense)", func() {
		BeforeEach(func() {
			stopAuctionBids = client.BidForStopAuction(RepAddressesFor("A", "RepThatErrors"), stopAuctionInfo)
		})

		It("does not return bids that (network) errored", func() {
			Ω(stopAuctionBids).Should(ConsistOf(auctiontypes.StopAuctionBids{
				{Rep: "A", InstanceGuids: []string{"A-1", "A-2"}, Bid: 0.27, Error: ""},
			}))
		})
	})
})
