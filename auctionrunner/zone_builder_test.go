package auctionrunner_test

import (
	"errors"
	"net/http"
	"time"

	"code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/auction/auctiontypes/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/rep"
	"code.cloudfoundry.org/rep/repfakes"
	"code.cloudfoundry.org/workpool"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"code.cloudfoundry.org/lager"
)

var _ = Describe("ZoneBuilder", func() {
	var repA, repB, repC *repfakes.FakeSimClient
	var clients map[string]rep.Client
	var workPool *workpool.WorkPool
	var logger *lagertest.TestLogger
	var metricEmitter *fakes.FakeAuctionMetricEmitterDelegate

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		var err error
		workPool, err = workpool.NewWorkPool(5)
		Expect(err).NotTo(HaveOccurred())

		repA = new(repfakes.FakeSimClient)
		repB = new(repfakes.FakeSimClient)
		repC = new(repfakes.FakeSimClient)

		clients = map[string]rep.Client{
			"A": repA,
			"B": repB,
			"C": repC,
		}

		repA.StateReturns(BuildCellState("A", "the-zone", 100, 200, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), nil)
		repB.StateReturns(BuildCellState("B", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), nil)
		repC.StateReturns(BuildCellState("C", "other-zone", 100, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), nil)

		metricEmitter = new(fakes.FakeAuctionMetricEmitterDelegate)
	})

	AfterEach(func() {
		workPool.Stop()
	})

	It("fetches state by calling each client", func() {
		zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)
		Expect(zones).To(HaveLen(2))

		cells := map[string]*auctionrunner.Cell{}
		for _, cell := range zones["the-zone"] {
			cells[cell.Guid] = cell
		}
		Expect(cells).To(HaveLen(2))
		Expect(cells).To(HaveKey("A"))
		Expect(cells).To(HaveKey("B"))

		Expect(repA.StateCallCount()).To(Equal(1))
		Expect(repB.StateCallCount()).To(Equal(1))

		otherZone := zones["other-zone"]
		Expect(otherZone).To(HaveLen(1))
		Expect(otherZone[0].Guid).To(Equal("C"))

		Expect(repC.StateCallCount()).To(Equal(1))
	})

	Context("when cells are evacuating", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState("B", "the-zone", 10, 10, 100, true, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), nil)
		})

		It("does not include them in the map", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)
			Expect(zones).To(HaveLen(2))

			cells := zones["the-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("A"))

			cells = zones["other-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("C"))
		})
	})

	Context("when a cell ID does not match the cell state ID", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState("badCellID", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), nil)
		})

		It("does not include that cell in the map", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)
			Expect(zones).To(HaveLen(2))

			cells := zones["the-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("A"))

			cells = zones["other-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("C"))
		})

		It("logs that there was a cell ID mismatch", func() {
			auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)

			getLogData := func(format lager.LogFormat) lager.Data { return format.Data }

			Expect(logger.LogMessages()).To(ContainElement("test.cell-id-mismatch"))
			Expect(logger.Logs()).To(ContainElement(WithTransform(getLogData, Equal(lager.Data{"cell-guid": "B", "cell-state-guid": "badCellID"}))))
		})
	})

	Context("when a client fails", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState("B", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), errors.New("boom"))
		})

		It("does not include the client in the map", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)
			Expect(zones).To(HaveLen(2))

			cells := zones["the-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("A"))

			cells = zones["other-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("C"))
		})

		It("it emits metrics for the failure", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients, metricEmitter)
			Expect(zones).To(HaveLen(2))
			Expect(metricEmitter.FailedCellStateRequestCallCount()).To(Equal(1))
		})
	})

	Context("when clients are slow to respond", func() {
		BeforeEach(func() {
			repA.StateReturns(BuildCellState("A", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), errors.New("timeout"))
			repA.StateClientTimeoutReturns(5 * time.Second)
			repA.SetStateClientStub = func(client *http.Client) {
				repA.StateClientTimeoutReturns(client.Timeout)
			}
			repB.StateReturns(BuildCellState("B", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), errors.New("timeout"))
			repB.StateClientTimeoutReturns(2 * time.Second)
			repB.SetStateClientStub = func(client *http.Client) {
				repB.StateClientTimeoutReturns(client.Timeout)
			}
			repC.StateReturns(BuildCellState("C", "the-zone", 10, 10, 100, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{}), errors.New("timeout"))
			repC.StateClientTimeoutReturns(4 * time.Second)
			repC.SetStateClientStub = func(client *http.Client) {
				repC.StateClientTimeoutReturns(client.Timeout)
			}
		})
	})
})
