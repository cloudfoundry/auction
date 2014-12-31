package auctionrunner_test

import (
	"errors"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry/gunk/workpool"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ZoneBuilder", func() {
	var repA, repB, repC *fakes.FakeSimulationCellRep
	var clients map[string]auctiontypes.CellRep
	var workPool *workpool.WorkPool

	BeforeEach(func() {
		workPool = workpool.NewWorkPool(5)
		repA = &fakes.FakeSimulationCellRep{}
		repB = &fakes.FakeSimulationCellRep{}
		repC = &fakes.FakeSimulationCellRep{}

		clients = map[string]auctiontypes.CellRep{
			"A": repA,
			"B": repB,
			"C": repC,
		}

		repA.StateReturns(BuildCellState("the-zone", 100, 200, 100, nil), nil)
		repB.StateReturns(BuildCellState("the-zone", 10, 10, 100, nil), nil)
		repC.StateReturns(BuildCellState("other-zone", 100, 10, 100, nil), nil)
	})

	AfterEach(func() {
		workPool.Stop()
	})

	It("fetches state by calling each client", func() {
		zones := FetchStateAndBuildZones(workPool, clients)
		Ω(zones).Should(HaveLen(2))

		cells := map[string]*Cell{}
		for _, cell := range zones["the-zone"] {
			cells[cell.Guid] = cell
		}
		Ω(cells).Should(HaveLen(2))
		Ω(cells).Should(HaveKey("A"))
		Ω(cells).Should(HaveKey("B"))

		Ω(repA.StateCallCount()).Should(Equal(1))
		Ω(repB.StateCallCount()).Should(Equal(1))

		otherZone := zones["other-zone"]
		Ω(otherZone).Should(HaveLen(1))
		Ω(otherZone[0].Guid).Should(Equal("C"))

		Ω(repC.StateCallCount()).Should(Equal(1))
	})

	Context("when a client fails", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState("the-zone", 10, 10, 100, nil), errors.New("boom"))
		})

		It("does not include the client in the map", func() {
			zones := FetchStateAndBuildZones(workPool, clients)
			Ω(zones).Should(HaveLen(2))

			cells := zones["the-zone"]
			Ω(cells).Should(HaveLen(1))
			Ω(cells[0].Guid).Should(Equal("A"))
		})
	})
})
