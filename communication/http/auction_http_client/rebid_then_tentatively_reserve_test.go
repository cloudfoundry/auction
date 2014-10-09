package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RebidThenTentativelyReserve", func() {
	var lrpStartAuction models.LRPStartAuction
	var startAuctionBids auctiontypes.StartAuctionBids

	BeforeEach(func() {
		lrpStartAuction = models.LRPStartAuction{
			InstanceGuid: "instance-guid",
			Index:        1,
		}

		auctionRepA.RebidThenTentativelyReserveReturns(0.27, nil)
	})

	It("should request reservations from all passed in reps", func() {
		client.RebidThenTentativelyReserve(RepAddressesFor("A", "B"), lrpStartAuction)

		Ω(auctionRepA.RebidThenTentativelyReserveCallCount()).Should(Equal(1))
		Ω(auctionRepA.RebidThenTentativelyReserveArgsForCall(0)).Should(Equal(lrpStartAuction))

		Ω(auctionRepB.RebidThenTentativelyReserveCallCount()).Should(Equal(1))
		Ω(auctionRepB.RebidThenTentativelyReserveArgsForCall(0)).Should(Equal(lrpStartAuction))
	})

	Context("when the reservations are succesful", func() {
		BeforeEach(func() {
			auctionRepB.RebidThenTentativelyReserveReturns(0.48, nil)
			startAuctionBids = client.RebidThenTentativelyReserve(RepAddressesFor("A", "B"), lrpStartAuction)
		})

		It("returns all bids", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
				{Rep: "B", Bid: 0.48, Error: ""},
			}))
		})
	})

	Context("when reservations are unsuccesful", func() {
		BeforeEach(func() {
			auctionRepB.RebidThenTentativelyReserveReturns(0, errors.New("oops"))
			startAuctionBids = client.RebidThenTentativelyReserve(RepAddressesFor("A", "B"), lrpStartAuction)
		})

		It("does not return them", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request doesn't succeed", func() {
		BeforeEach(func() {
			startAuctionBids = client.RebidThenTentativelyReserve(RepAddressesFor("A", "RepThat500s"), lrpStartAuction)
		})

		It("does not return bids that didn't succeed", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})

	Context("when a request errors (in the network sense)", func() {
		BeforeEach(func() {
			startAuctionBids = client.RebidThenTentativelyReserve(RepAddressesFor("A", "RepThatErrors"), lrpStartAuction)
		})

		It("does not return bids that (network) errored", func() {
			Ω(startAuctionBids).Should(ConsistOf(auctiontypes.StartAuctionBids{
				{Rep: "A", Bid: 0.27, Error: ""},
			}))
		})
	})
})
