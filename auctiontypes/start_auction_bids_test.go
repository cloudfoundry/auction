package auctiontypes_test

import (
	. "github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartAuctionBids", func() {
	var bids StartAuctionBids
	BeforeEach(func() {
		bids = StartAuctionBids{
			{Rep: "A", Bid: 0.7, Error: ""},
			{Rep: "B", Bid: 1.2, Error: ""},
			{Rep: "C", Bid: 0, Error: "owie"},
			{Rep: "D", Bid: 0.9, Error: ""},
		}
	})

	Describe("AllFailed", func() {
		It("should be true when all bids have failed", func() {
			bids = StartAuctionBids{
				{Rep: "A", Bid: 0, Error: "ouch"},
				{Rep: "C", Bid: 0, Error: "owie"},
			}

			Ω(bids.AllFailed()).Should(BeTrue())
		})

		It("should be false if some bids have not failed", func() {
			Ω(bids.AllFailed()).Should(BeFalse())
		})
	})

	Describe("Reps", func() {
		It("should return the rep guids, in order", func() {
			Ω(bids.Reps()).Should(Equal([]string{"A", "B", "C", "D"}))
		})
	})

	Describe("FilterErrors", func() {
		It("should return the bids that did not error", func() {
			Ω(bids.FilterErrors()).Should(Equal(StartAuctionBids{bids[0], bids[1], bids[3]}))
		})
	})

	Describe("Shuffle", func() {
		It("should shuffle the order of the bids", func() {
			Ω(bids.Shuffle()).Should(ConsistOf(bids))
			Eventually(bids.Shuffle).ShouldNot(Equal(bids))
		})
	})

	Describe("Sort", func() {
		It("should sort the bids", func() {
			sorted := StartAuctionBids{bids[2], bids[0], bids[3], bids[1]}
			Ω(bids.Sort()).Should(Equal(sorted))
		})
	})
})
