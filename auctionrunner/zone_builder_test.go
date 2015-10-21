package auctionrunner_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry-incubator/rep/repfakes"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ZoneBuilder", func() {
	var repA, repB, repC *repfakes.FakeSimClient
	var clients map[string]rep.Client
	var workPool *workpool.WorkPool
	var logger lager.Logger

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

		repA.StateReturns(BuildCellState("the-zone", 100, 200, 100, false, linuxOnlyRootFSProviders, nil), nil)
		repB.StateReturns(BuildCellState("the-zone", 10, 10, 100, false, linuxOnlyRootFSProviders, nil), nil)
		repC.StateReturns(BuildCellState("other-zone", 100, 10, 100, false, linuxOnlyRootFSProviders, nil), nil)
	})

	AfterEach(func() {
		workPool.Stop()
	})

	It("fetches state by calling each client", func() {
		zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients)
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
			repB.StateReturns(BuildCellState("the-zone", 10, 10, 100, true, linuxOnlyRootFSProviders, nil), nil)
		})

		It("does not include them in the map", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients)
			Expect(zones).To(HaveLen(2))

			cells := zones["the-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("A"))

			cells = zones["other-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("C"))
		})
	})

	Context("when a client fails", func() {
		BeforeEach(func() {
			repB.StateReturns(BuildCellState("the-zone", 10, 10, 100, false, linuxOnlyRootFSProviders, nil), errors.New("boom"))
		})

		It("does not include the client in the map", func() {
			zones := auctionrunner.FetchStateAndBuildZones(logger, workPool, clients)
			Expect(zones).To(HaveLen(2))

			cells := zones["the-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("A"))

			cells = zones["other-zone"]
			Expect(cells).To(HaveLen(1))
			Expect(cells[0].Guid).To(Equal("C"))
		})
	})
})
