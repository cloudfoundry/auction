package auctionrep_test

import (
	"errors"

	. "github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Resources(diskMB int, memoryMB int, containers int) auctiontypes.Resources {
	return auctiontypes.Resources{
		DiskMB:     diskMB,
		MemoryMB:   memoryMB,
		Containers: containers,
	}
}

var _ = Describe("Auction Rep", func() {
	var delegate *fakes.FakeAuctionRepDelegate
	var rep *AuctionRep
	var startAuctionInfo auctiontypes.StartAuctionInfo
	var stopAuctionInfo auctiontypes.StopAuctionInfo

	BeforeEach(func() {
		delegate = &fakes.FakeAuctionRepDelegate{}
		rep = New("rep-guid", delegate)

		startAuctionInfo = auctiontypes.StartAuctionInfo{
			ProcessGuid:  "process-guid",
			InstanceGuid: "instance-guid",
			DiskMB:       10,
			MemoryMB:     20,
			Index:        0,
		}

		stopAuctionInfo = auctiontypes.StopAuctionInfo{
			ProcessGuid: "process-guid",
			Index:       3,
		}
	})

	Describe("Guid", func() {
		It("should return the guid it was given at birth", func() {
			Ω(rep.Guid()).Should(Equal("rep-guid"))
		})
	})

	Describe("BidForStartAuction", func() {
		ItShouldComputeStartBidsCorrectly(func() *fakes.FakeAuctionRepDelegate {
			return delegate
		}, func() StartAuctionBidFunc {
			return rep.BidForStartAuction
		})
	})

	Describe("RebidThenTentativelyReserve", func() {
		ItShouldComputeStartBidsCorrectly(func() *fakes.FakeAuctionRepDelegate {
			return delegate
		}, func() StartAuctionBidFunc {
			return rep.RebidThenTentativelyReserve
		})

		Describe("Reserving", func() {

			BeforeEach(func() {
				delegate.RemainingResourcesReturns(Resources(startAuctionInfo.DiskMB, startAuctionInfo.MemoryMB, 1), nil)
				delegate.TotalResourcesReturns(Resources(startAuctionInfo.DiskMB*2, startAuctionInfo.MemoryMB*2, 2), nil)
			})

			It("should make the reservation after computing the scores", func() {
				//It's important that the reservation happen after the score is computed.
				//In principal, with a real delegate, this test could be written behaviorally.
				//Instead we do the mocky mocky stub stub.  The auction simulation will degrade if the ordering is flipped.
				ordering := []string{}

				delegate.TotalResourcesStub = func() (auctiontypes.Resources, error) {
					ordering = append(ordering, "Total")
					return Resources(startAuctionInfo.DiskMB, startAuctionInfo.MemoryMB, 1), nil
				}

				delegate.RemainingResourcesStub = func() (auctiontypes.Resources, error) {
					ordering = append(ordering, "Remaining")
					return Resources(startAuctionInfo.DiskMB, startAuctionInfo.MemoryMB, 1), nil
				}

				delegate.ReserveStub = func(startAuctionInfo auctiontypes.StartAuctionInfo) error {
					ordering = append(ordering, "Reserve")
					return nil
				}

				_, err := rep.RebidThenTentativelyReserve(startAuctionInfo)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(ordering).Should(Equal([]string{"Remaining", "Total", "Reserve"}))
			})

			It("should pass the reservation the correct startAuctionInfo", func() {
				_, err := rep.RebidThenTentativelyReserve(startAuctionInfo)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(delegate.ReserveArgsForCall(0)).Should(Equal(startAuctionInfo))
			})

			Context("when the reservation fails", func() {
				It("should error", func() {
					delegate.ReserveReturns(errors.New("kaboom"))
					_, err := rep.RebidThenTentativelyReserve(startAuctionInfo)
					Ω(err).Should(MatchError(errors.New("kaboom")))
				})
			})
		})
	})

	Describe("ReleaseReservation", func() {
		It("should instruct the delegate to release the reservation", func() {
			rep.ReleaseReservation(startAuctionInfo)
			Ω(delegate.ReleaseReservationArgsForCall(0)).Should(Equal(startAuctionInfo))
		})

		Context("when the delegate errors", func() {
			It("should error", func() {
				delegate.ReleaseReservationReturns(errors.New("kaboom"))
				err := rep.ReleaseReservation(startAuctionInfo)
				Ω(err).Should(MatchError(errors.New("kaboom")))
			})
		})
	})

	Describe("Run", func() {
		var startAuction models.LRPStartAuction

		BeforeEach(func() {
			startAuction = models.LRPStartAuction{
				InstanceGuid: "instance-guid",
			}
		})

		It("should instruct the delegate to run", func() {
			rep.Run(startAuction)
			Ω(delegate.RunArgsForCall(0)).Should(Equal(startAuction))
		})

		Context("when the delegate errors", func() {
			It("should error", func() {
				delegate.RunReturns(errors.New("kaboom"))
				err := rep.Run(startAuction)
				Ω(err).Should(MatchError(errors.New("kaboom")))
			})
		})
	})

	Describe("BidForStopAuction", func() {
		It("should ask the delegate for instance guids", func() {
			rep.BidForStopAuction(stopAuctionInfo)
			processGuid, index := delegate.InstanceGuidsForProcessGuidAndIndexArgsForCall(0)
			Ω(processGuid).Should(Equal(stopAuctionInfo.ProcessGuid))
			Ω(index).Should(Equal(stopAuctionInfo.Index))
		})

		Context("when the delegate fails to provide resource information", func() {
			BeforeEach(func() {
				delegate.TotalResourcesReturns(Resources(0, 0, 0), errors.New("kaboom"))
			})

			It("should error", func() {
				_, _, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).Should(MatchError(errors.New("kaboom")))
			})
		})

		Context("when the delegate fails to provide a response for instance guids for the given processGuid and Index", func() {
			BeforeEach(func() {
				delegate.InstanceGuidsForProcessGuidAndIndexReturns(nil, errors.New("kaboom"))
			})

			It("should error", func() {
				_, _, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).Should(MatchError(errors.New("kaboom")))
			})
		})

		Context("when the delegate is not running any instance guids for the given processGuid and index", func() {
			BeforeEach(func() {
				delegate.InstanceGuidsForProcessGuidAndIndexReturns([]string{}, nil)
			})

			It("should error", func() {
				_, _, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).Should(MatchError(auctiontypes.NothingToStop))
			})
		})

		Context("when the delegate is running instance guids", func() {
			BeforeEach(func() {
				delegate.InstanceGuidsForProcessGuidAndIndexReturns([]string{"a", "b", "c"}, nil)
			})

			It("should return the instance guids", func() {
				_, instanceGuids, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(instanceGuids).Should(Equal([]string{"a", "b", "c"}))
			})

			It("should return a lower score for delegates with higher resource availability", func() {
				// fractionAvailable = 1/3
				delegate.RemainingResourcesReturns(Resources(20, 10, 1), nil)
				delegate.TotalResourcesReturns(Resources(20*3, 10*2, 2), nil)

				scoreA, _, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).ShouldNot(HaveOccurred())

				// fractionAvailable = 1/2
				delegate.RemainingResourcesReturns(Resources(20, 10, 1), nil)
				delegate.TotalResourcesReturns(Resources(20*2, 10*2, 2), nil)

				scoreB, _, err := rep.BidForStopAuction(stopAuctionInfo)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(scoreB).Should(BeNumerically("<", scoreA))
			})
		})
	})

	Describe("Stop", func() {
		var stopInstance models.StopLRPInstance

		BeforeEach(func() {
			stopInstance = models.StopLRPInstance{
				InstanceGuid: "instance-guid",
			}
		})

		It("should instruct the delegate to run", func() {
			rep.Stop(stopInstance)
			Ω(delegate.StopArgsForCall(0)).Should(Equal(stopInstance))
		})

		Context("when the delegate errors", func() {
			It("should error", func() {
				delegate.StopReturns(errors.New("kaboom"))
				err := rep.Stop(stopInstance)
				Ω(err).Should(MatchError(errors.New("kaboom")))
			})
		})
	})

	Describe("Locking around actions", func() {
		It("should serialize calls to BidForStartAuction", func() {
			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.TotalResourcesStub = func() (auctiontypes.Resources, error) {
				spy <- struct{}{}
				<-release
				return Resources(startAuctionInfo.DiskMB, startAuctionInfo.MemoryMB, 1), nil
			}

			go rep.BidForStartAuction(startAuctionInfo)
			go rep.BidForStartAuction(startAuctionInfo)

			Eventually(spy).Should(Receive())
			Consistently(spy).ShouldNot(Receive())
			release <- struct{}{}
			Eventually(spy).Should(Receive())
		})

		It("should serialize calls to BidForStopAuction", func() {
			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.TotalResourcesStub = func() (auctiontypes.Resources, error) {
				spy <- struct{}{}
				<-release
				return Resources(1, 1, 1), nil
			}

			go rep.BidForStopAuction(stopAuctionInfo)
			go rep.BidForStopAuction(stopAuctionInfo)

			Eventually(spy).Should(Receive())
			Consistently(spy).ShouldNot(Receive())
			release <- struct{}{}
			Eventually(spy).Should(Receive())
		})

		It("should serialize calls to RebidThenTentativelyReserve", func() {
			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.TotalResourcesStub = func() (auctiontypes.Resources, error) {
				spy <- struct{}{}
				<-release
				return Resources(startAuctionInfo.DiskMB, startAuctionInfo.MemoryMB, 1), nil
			}

			go rep.RebidThenTentativelyReserve(startAuctionInfo)
			go rep.RebidThenTentativelyReserve(startAuctionInfo)

			Eventually(spy).Should(Receive())
			Consistently(spy).ShouldNot(Receive())
			release <- struct{}{}
			Eventually(spy).Should(Receive())
			close(release)
		})

		It("should serialize calls to ReleaseReservation", func() {
			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.ReleaseReservationStub = func(auctiontypes.StartAuctionInfo) error {
				spy <- struct{}{}
				<-release
				return nil
			}

			go rep.ReleaseReservation(startAuctionInfo)
			go rep.ReleaseReservation(startAuctionInfo)

			Eventually(spy).Should(Receive())
			Consistently(spy).ShouldNot(Receive())
			release <- struct{}{}
			Eventually(spy).Should(Receive())
			close(release)
		})

		It("should *not* serialize calls to Run", func() {
			startAuction := models.LRPStartAuction{
				InstanceGuid: "instance-guid",
			}

			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.RunStub = func(models.LRPStartAuction) error {
				spy <- struct{}{}
				<-release
				return nil
			}

			go rep.Run(startAuction)
			go rep.Run(startAuction)

			Eventually(spy).Should(Receive())
			Eventually(spy).Should(Receive())
			close(release)
		})

		It("should *not* serialize calls to Stop", func() {
			stopInstance := models.StopLRPInstance{
				InstanceGuid: "instance-guid",
			}

			spy := make(chan struct{}, 2)
			release := make(chan struct{})

			delegate.StopStub = func(models.StopLRPInstance) error {
				spy <- struct{}{}
				<-release
				return nil
			}

			go rep.Stop(stopInstance)
			go rep.Stop(stopInstance)

			Eventually(spy).Should(Receive())
			Eventually(spy).Should(Receive())
			close(release)
		})
	})
})
