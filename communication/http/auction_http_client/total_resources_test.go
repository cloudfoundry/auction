package auction_http_client_test

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) TotalResources", func() {
	var totalResources auctiontypes.Resources

	BeforeEach(func() {
		totalResources = auctiontypes.Resources{
			MemoryMB:   1024,
			DiskMB:     512,
			Containers: 64,
		}

		auctionRepA.TotalResourcesReturns(totalResources)
	})

	It("should return the resources returned by the rep", func() {
		returnedResources := client.TotalResources(RepAddressFor("A"))
		Ω(returnedResources).Should(Equal(totalResources))
	})

	Context("when a request doesn't succeed", func() {
		It("does not return any resources", func() {
			returnedResources := client.TotalResources(RepAddressFor("RepThat500s"))
			Ω(returnedResources).Should(BeZero())
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("does not return any resources", func() {
			returnedResources := client.TotalResources(RepAddressFor("RepThatErrors"))
			Ω(returnedResources).Should(BeZero())
		})
	})
})
