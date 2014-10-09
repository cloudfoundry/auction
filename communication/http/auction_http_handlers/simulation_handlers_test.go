package auction_http_handlers_test

import (
	"bytes"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SimulationHandlers", func() {
	var resources auctiontypes.Resources
	var simulatedInstances []auctiontypes.SimulatedInstance

	BeforeEach(func() {
		resources = auctiontypes.Resources{
			MemoryMB:   1024,
			DiskMB:     512,
			Containers: 256,
		}

		simulatedInstances = []auctiontypes.SimulatedInstance{
			{ProcessGuid: "A", InstanceGuid: "Alpha", Index: 1, MemoryMB: 1024, DiskMB: 256},
			{ProcessGuid: "B", InstanceGuid: "Beta", Index: 2, MemoryMB: 512, DiskMB: 128},
		}
	})

	Describe("Getting Total Resources", func() {
		BeforeEach(func() {
			auctionRep.TotalResourcesReturns(resources)
		})

		It("returns the resources returned by the rep", func() {
			status, body := Request(routes.Sim_TotalResources, nil, nil)
			Ω(status).Should(Equal(http.StatusOK))
			Ω(body).Should(MatchJSON(JSONFor(resources)))
		})
	})

	Describe("Reset", func() {
		It("tells the rep to reset", func() {
			Ω(auctionRep.ResetCallCount()).Should(Equal(0))

			status, body := Request(routes.Sim_Reset, nil, nil)
			Ω(status).Should(Equal(http.StatusOK))
			Ω(body).Should(BeEmpty())

			Ω(auctionRep.ResetCallCount()).Should(Equal(1))
		})
	})

	Describe("Getting Simulated Instances", func() {
		BeforeEach(func() {
			auctionRep.SimulatedInstancesReturns(simulatedInstances)
		})

		It("returns the simulated instances returned by the rep", func() {
			status, body := Request(routes.Sim_SimulatedInstances, nil, nil)
			Ω(status).Should(Equal(http.StatusOK))
			Ω(body).Should(MatchJSON(JSONFor(simulatedInstances)))
		})
	})

	Describe("Setting Simulated Instances", func() {
		Context("when passed valid json", func() {
			It("tells the rep to set simulated instances", func() {
				status, body := Request(routes.Sim_SetSimulatedInstances, nil, JSONReaderFor(simulatedInstances))
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(BeEmpty())

				Ω(auctionRep.SetSimulatedInstancesCallCount()).Should(Equal(1))
				Ω(auctionRep.SetSimulatedInstancesArgsForCall(0)).Should(Equal(simulatedInstances))
			})
		})

		Context("when passed invalid json", func() {
			It("errors", func() {
				status, body := Request(routes.Sim_SetSimulatedInstances, nil, bytes.NewBufferString("∆"))
				Ω(status).Should(Equal(http.StatusBadRequest))
				Ω(body).Should(ContainSubstring("invalid json: invalid character"))
			})
		})
	})
})
