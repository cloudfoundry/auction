package auction_http_client_test

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) SimulatedInstances", func() {
	var instances []auctiontypes.SimulatedInstance

	BeforeEach(func() {
		instances = []auctiontypes.SimulatedInstance{
			{
				ProcessGuid:  "process-guid-1",
				InstanceGuid: "instance-guid-1",
				Index:        3,
				MemoryMB:     1024,
				DiskMB:       512,
			},
			{
				ProcessGuid:  "process-guid-2",
				InstanceGuid: "instance-guid-2",
				Index:        4,
				MemoryMB:     512,
				DiskMB:       256,
			},
		}

		auctionRepA.SimulatedInstancesReturns(instances)
	})

	It("should return the instances returned by the rep", func() {
		returnedInstances := client.SimulatedInstances(RepAddressFor("A"))
		Ω(returnedInstances).Should(Equal(instances))
	})

	Context("when a request doesn't succeed", func() {
		It("does not return any instances", func() {
			returnedInstances := client.SimulatedInstances(RepAddressFor("RepThat500s"))
			Ω(returnedInstances).Should(BeNil())
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("does not return any instances", func() {
			returnedInstances := client.SimulatedInstances(RepAddressFor("RepThatErrors"))
			Ω(returnedInstances).Should(BeNil())
		})
	})
})
