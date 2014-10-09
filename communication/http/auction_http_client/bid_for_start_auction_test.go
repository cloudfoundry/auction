package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BidForStartAuction", func() {
	var startAuctionInfo auctiontypes.StartAuctionInfo
	var startAuctionBids auctiontypes.StartAuctionBids

	BeforeEach(func() {
		startAuctionInfo = auctiontypes.StartAuctionInfo{
			ProcessGuid:  "process-guid",
			InstanceGuid: "instance-guid",
			Index:        1,
			DiskMB:       1024,
			MemoryMB:     256,
		}

		auctionRepA.BidForStartAuctionReturns(0.27, nil)
	})

	It("should request bids from all passed in reps", func() {
		client.BidForStartAuction(RepAddressesFor("A", "B"), startAuctionInfo)

		Ω(auctionRepA.BidForStartAuctionCallCount()).Should(Equal(1))
		Ω(auctionRepA.BidForStartAuctionArgsForCall(0)).Should(Equal(startAuctionInfo))

		Ω(auctionRepB.BidForStartAuctionCallCount()).Should(Equal(1))
		Ω(auctionRepB.BidForStartAuctionArgsForCall(0)).Should(Equal(startAuctionInfo))
	})

	Context("when the bids are succesful", func() {
		BeforeEach(func() {
			auctionRepB.BidForStartAuctionReturns(0.48, nil)
			startAuctionBids = client.BidForStartAuction(RepAddressesFor("A", "B"), startAuctionInfo)
		})

		It("returns all bids", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
				{Rep: "B", Bid: 0.48, Error: ""},
			}))
		})
	})

	Context("when bids are unsuccesful", func() {
		BeforeEach(func() {
			auctionRepB.BidForStartAuctionReturns(0, errors.New("oops"))
			startAuctionBids = client.BidForStartAuction(RepAddressesFor("A", "B"), startAuctionInfo)
		})

		It("does not return them", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request doesn't succeed", func() {
		BeforeEach(func() {
			startAuctionBids = client.BidForStartAuction(RepAddressesFor("A", "RepThat500s"), startAuctionInfo)
		})

		It("does not return bids that didn't succeed", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request errors (in the network sense)", func() {
		BeforeEach(func() {
			startAuctionBids = client.BidForStartAuction(RepAddressesFor("A", "RepThatErrors"), startAuctionInfo)
		})

		It("does not return bids that (network) errored", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})
})
