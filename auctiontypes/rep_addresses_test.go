package auctiontypes_test

import (
	. "github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RepAddresses", func() {
	var addresses RepAddresses

	BeforeEach(func() {
		addresses = RepAddresses{
			{"A", "address-A"},
			{"B", "address-B"},
			{"C", "address-C"},
		}
	})

	Describe("RandomSubsetByCount", func() {
		Context("when the desired count is less than the number of addresses", func() {
			It("should return a random subset", func() {
				subset := addresses.RandomSubsetByCount(2)
				Ω(subset).Should(HaveLen(2))
				Ω(addresses).Should(ContainElement(subset[0]))
				Ω(addresses).Should(ContainElement(subset[1]))
				Ω(subset[0]).ShouldNot(Equal(subset[1]))
			})
		})

		Context("when the desired count exceeds the number of addresses", func() {
			It("should return all addresses", func() {
				Ω(addresses.RandomSubsetByCount(4)).Should(Equal(addresses))
			})
		})
	})

	Describe("Random subset by fraction", func() {
		Context("when passed a fraction that would exceed the minimum number", func() {
			It("should return that fraction of addresses", func() {
				Ω(addresses.RandomSubsetByFraction(0.5, 1)).Should(HaveLen(2))
			})
		})
		Context("when the fraction would be less than the minimum number", func() {
			It("should return the minimum number", func() {
				Ω(addresses.RandomSubsetByFraction(0.1, 1)).Should(HaveLen(1))
			})
		})

		Context("when passed a fraction exceeding 1", func() {
			It("should return all addresses", func() {
				Ω(addresses.RandomSubsetByFraction(1.1, 1)).Should(Equal(addresses))
			})
		})
	})

	Describe("Without", func() {
		It("should return a copy of the addresses without the passed-in rep guids", func() {
			Ω(addresses.Without("A", "D")).Should(Equal(addresses[1:]))
		})
	})

	Describe("Lookup", func() {
		It("should return a addresses that match the passed in Guids", func() {
			subset := addresses.Lookup("B", "D", "A")
			Ω(subset).Should(Equal(addresses[0:2]))
		})
	})

	Describe("AddressFor", func() {
		It("should return the address for the given guid", func() {
			Ω(addresses.AddressFor("B")).Should(Equal(addresses[1]))
		})

		It("should return the zero value if the guid is not present", func() {
			Ω(addresses.AddressFor("D")).Should(BeZero())
		})
	})
})
