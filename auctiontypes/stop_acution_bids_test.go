package auctiontypes_test

import (
	. "github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopAuctionBids", func() {
	var bids StopAuctionBids
	BeforeEach(func() {
		bids = StopAuctionBids{
			{Rep: "A", InstanceGuids: []string{"A-1", "A-2"}, Bid: 0.7, Error: ""},
			{Rep: "B", InstanceGuids: []string{"B-1", "B-2"}, Bid: 1.2, Error: ""},
			{Rep: "C", InstanceGuids: nil, Bid: 0, Error: "owie"},
			{Rep: "D", InstanceGuids: []string{"D-1", "D-2"}, Bid: 0.9, Error: ""},
		}
	})

	Describe("AllFailed", func() {
		It("should be true when all bids have failed", func() {
			bids = StopAuctionBids{
				{Rep: "A", InstanceGuids: nil, Bid: 0, Error: "ouch"},
				{Rep: "C", InstanceGuids: nil, Bid: 0, Error: "owie"},
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

	Describe("InstanceGuids", func() {
		It("should return all instance guids concatenated in order", func() {
			Ω(bids.InstanceGuids()).Should(Equal([]string{"A-1", "A-2", "B-1", "B-2", "D-1", "D-2"}))
		})
	})

	Describe("FilterErrors", func() {
		It("should return the bids that did not error", func() {
			Ω(bids.FilterErrors()).Should(Equal(StopAuctionBids{bids[0], bids[1], bids[3]}))
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
			sorted := StopAuctionBids{bids[2], bids[0], bids[3], bids[1]}
			Ω(bids.Sort()).Should(Equal(sorted))
		})
	})
})
