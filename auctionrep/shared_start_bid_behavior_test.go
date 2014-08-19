package auctionrep_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
)

type StartAuctionBidFunc func(auctiontypes.StartAuctionInfo) (float64, error)

func ItShouldComputeStartBidsCorrectly(delegateFetcher func() *fakes.FakeAuctionRepDelegate, bidFuncFetcher func() StartAuctionBidFunc) {
	Describe("[Shared Examples] Computing start bids", func() {
		var delegate *fakes.FakeAuctionRepDelegate
		var bidFunc StartAuctionBidFunc

		var startAuctionInfo auctiontypes.StartAuctionInfo
		var fixtureError error
		var diskMB, memoryMB int

		BeforeEach(func() {
			delegate = delegateFetcher()
			bidFunc = bidFuncFetcher()

			diskMB = 10
			memoryMB = 20
			startAuctionInfo = auctiontypes.StartAuctionInfo{
				ProcessGuid:  "process-guid",
				InstanceGuid: "instance-guid",
				DiskMB:       diskMB,
				MemoryMB:     memoryMB,
				Index:        0,
			}

			fixtureError = errors.New("kaboom")
		})

		Context("when the delegate errors trying to fetch remaining resources", func() {
			BeforeEach(func() {
				delegate.RemainingResourcesReturns(Resources(0, 0, 0), fixtureError)
			})

			It("should error", func() {
				score, err := bidFunc(startAuctionInfo)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(fixtureError))
			})
		})

		Context("when the delegate errors trying to fetch total resources", func() {
			BeforeEach(func() {
				delegate.TotalResourcesReturns(Resources(0, 0, 0), fixtureError)
			})

			It("should error", func() {
				score, err := bidFunc(startAuctionInfo)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(fixtureError))
			})
		})

		Context("when the delegate errors trying to return the number of instances for the process guid", func() {
			BeforeEach(func() {
				delegate.NumInstancesForProcessGuidReturns(0, fixtureError)
			})

			It("should error", func() {
				score, err := bidFunc(startAuctionInfo)
				Ω(score).Should(BeZero())
				Ω(err).Should(MatchError(fixtureError))
				Ω(delegate.NumInstancesForProcessGuidArgsForCall(0)).Should(Equal(startAuctionInfo.ProcessGuid))
			})
		})

		Context("when the delegate has capacity (satisfies constraints)", func() {
			BeforeEach(func() {
				delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
				delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)
				delegate.NumInstancesForProcessGuidReturns(1, nil)
			})

			It("should return a score", func() {
				score, err := bidFunc(startAuctionInfo)
				Ω(score).ShouldNot(BeZero())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the delegate has no capacity (fails to satisfy constraints)", func() {
			Context("with insufficient disk", func() {
				BeforeEach(func() {
					delegate.RemainingResourcesReturns(Resources(diskMB-1, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)
					delegate.NumInstancesForProcessGuidReturns(1, nil)
				})

				It("should return an InsufficientResources error", func() {
					score, err := bidFunc(startAuctionInfo)
					Ω(score).Should(BeZero())
					Ω(err).Should(Equal(auctiontypes.InsufficientResources))
				})
			})

			Context("with insufficient memory", func() {
				BeforeEach(func() {
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB-1, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)
					delegate.NumInstancesForProcessGuidReturns(1, nil)
				})

				It("should return an InsufficientResources error", func() {
					score, err := bidFunc(startAuctionInfo)
					Ω(score).Should(BeZero())
					Ω(err).Should(Equal(auctiontypes.InsufficientResources))
				})
			})

			Context("with insufficient containers", func() {
				BeforeEach(func() {
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 0), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)
					delegate.NumInstancesForProcessGuidReturns(1, nil)
				})

				It("should return an InsufficientResources error", func() {
					score, err := bidFunc(startAuctionInfo)
					Ω(score).Should(BeZero())
					Ω(err).Should(Equal(auctiontypes.InsufficientResources))
				})
			})
		})

		Describe("the scores", func() {
			Context("all other things being equal", func() {
				It("should have a lower score for delegates with a higher fraction of available disk", func() {
					delegate.NumInstancesForProcessGuidReturns(1, nil)

					// fractionAvailable = 1/3
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*3, memoryMB*2, 2), nil)

					scoreA, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 1/2
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)

					scoreB, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 7/8
					delegate.RemainingResourcesReturns(Resources(diskMB*7, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*8, memoryMB*2, 2), nil)

					scoreC, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreC).Should(BeNumerically("<", scoreB))
					Ω(scoreB).Should(BeNumerically("<", scoreA))
				})

				It("should have a lower score for delegates with a higher fraction of available memory", func() {

					delegate.NumInstancesForProcessGuidReturns(1, nil)
					// fractionAvailable = 1/3
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*3, 2), nil)

					scoreA, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 1/2
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)

					scoreB, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 7/8
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB*7, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*8, 2), nil)

					scoreC, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreC).Should(BeNumerically("<", scoreB))
					Ω(scoreB).Should(BeNumerically("<", scoreA))
				})

				It("should have a lower score for delegates with a higher fraction of available containers", func() {
					delegate.NumInstancesForProcessGuidReturns(1, nil)

					// fractionAvailable = 1/3
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 3), nil)

					scoreA, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 1/2
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)

					scoreB, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// fractionAvailable = 7/8
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 7), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 8), nil)

					scoreC, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreC).Should(BeNumerically("<", scoreB))
					Ω(scoreB).Should(BeNumerically("<", scoreA))
				})

				It("should have a lower score for delegates not already running an instance for the given app", func() {
					delegate.RemainingResourcesReturns(Resources(diskMB, memoryMB, 1), nil)
					delegate.TotalResourcesReturns(Resources(diskMB*2, memoryMB*2, 2), nil)

					// running 2 instances of this app
					delegate.NumInstancesForProcessGuidReturns(2, nil)

					scoreA, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// running 1 instance of this app
					delegate.NumInstancesForProcessGuidReturns(1, nil)

					scoreB, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					// running no instances of this app
					delegate.NumInstancesForProcessGuidReturns(0, nil)

					scoreC, err := bidFunc(startAuctionInfo)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(scoreC).Should(BeNumerically("<", scoreB))
					Ω(scoreB).Should(BeNumerically("<", scoreA))
				})
			})
		})
	})
}
