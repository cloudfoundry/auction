package auctionrunner_test

import (
	"errors"
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry/gunk/workpool"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CellBuilder", func() {
	var repA, repB *fakes.FakeSimulationCellRep
	var clients map[string]auctiontypes.CellRep
	var workPool *workpool.WorkPool

	BeforeEach(func() {
		workPool = workpool.NewWorkPool(5)
		repA = &fakes.FakeSimulationCellRep{}
		repB = &fakes.FakeSimulationCellRep{}

		clients = map[string]auctiontypes.CellRep{
			"A": repA,
			"B": repB,
		}

		repA.StateReturns(BuildCellState(100, 200, 100, nil), nil)
		repB.StateReturns(BuildCellState(10, 10, 100, nil), nil)
	})

	AfterEach(func() {
		workPool.Stop()
	})

	It("fetches state by calling each client", func() {
		cells := FetchStateAndBuildCells(workPool, clients)
		Ω(cells).Should(HaveLen(2))
		Ω(cells).Should(HaveKey("A"))
		Ω(cells).Should(HaveKey("B"))

		_, err := cells["A"].ScoreForLRPAuction(BuildLRPAuction("pg-1", 0, "lucid64", 20, 20, time.Now()))
		Ω(err).ShouldNot(HaveOccurred())

		_, err = cells["B"].ScoreForLRPAuction(BuildLRPAuction("pg-1", 0, "lucid64", 20, 20, time.Now()))
		Ω(err).Should(MatchError(auctiontypes.ErrorInsufficientResources))
	})

	Context("when a client fails", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState(10, 10, 100, nil), errors.New("boom"))
		})

		It("does not include the client in the map", func() {
			cells := FetchStateAndBuildCells(workPool, clients)
			Ω(cells).Should(HaveLen(1))
			Ω(cells).Should(HaveKey("A"))
		})
	})
})
