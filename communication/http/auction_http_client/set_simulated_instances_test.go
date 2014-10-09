package auction_http_client_test

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("(Sim) SetSimulatedInstances", func() {
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
	})

	It("should post the instances to the rep", func() {
		client.SetSimulatedInstances(RepAddressFor("A"), instances)
		Ω(auctionRepA.SetSimulatedInstancesArgsForCall(0)).Should(Equal(instances))
	})

	Context("when a request doesn't succeed", func() {
		It("does not explode", func() {
			client.SetSimulatedInstances(RepAddressFor("RepThat500s"), instances)
		})
	})

	Context("when a request errors (in the network sense)", func() {
		It("does not explode", func() {
			client.SetSimulatedInstances(RepAddressFor("RepThatErrors"), instances)
		})
	})
})
