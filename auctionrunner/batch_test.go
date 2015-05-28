package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock/fakeclock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch", func() {
	var lrpStart models.LRPStartRequest
	var task models.Task
	var batch *auctionrunner.Batch
	var clock *fakeclock.FakeClock

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		batch = auctionrunner.NewBatch(clock)
	})

	It("should start off empty", func() {
		Expect(batch.HasWork).NotTo(Receive())
		starts, tasks := batch.DedupeAndDrain()
		Expect(starts).To(BeEmpty())
		Expect(tasks).To(BeEmpty())
	})

	Describe("adding work", func() {
		Context("when adding start auctions", func() {
			BeforeEach(func() {
				lrpStart = BuildLRPStartRequest("pg-1", []uint{1}, "linux", 10, 10)
				batch.AddLRPStarts([]models.LRPStartRequest{lrpStart})
			})

			It("makes the start auction available when drained", func() {
				lrpAuctions, _ := batch.DedupeAndDrain()
				Expect(lrpAuctions).To(ConsistOf(BuildLRPAuctions(lrpStart, clock.Now())))
			})

			It("should have work", func() {
				Expect(batch.HasWork).To(Receive())
			})
		})

		Context("when adding tasks", func() {
			BeforeEach(func() {
				task = BuildTask("tg-1", "linux", 10, 10)
				batch.AddTasks([]models.Task{task})
			})

			It("makes the stop auction available when drained", func() {
				_, taskAuctions := batch.DedupeAndDrain()
				Expect(taskAuctions).To(ConsistOf(BuildTaskAuction(task, clock.Now())))
			})

			It("should have work", func() {
				Expect(batch.HasWork).To(Receive())
			})
		})
	})

	Describe("DedupeAndDrain", func() {
		BeforeEach(func() {
			batch.AddLRPStarts([]models.LRPStartRequest{
				BuildLRPStartRequest("pg-1", []uint{1}, "linux", 10, 10),
				BuildLRPStartRequest("pg-1", []uint{1}, "linux", 10, 10),
				BuildLRPStartRequest("pg-2", []uint{2}, "linux", 10, 10),
			})

			batch.AddTasks([]models.Task{
				BuildTask("tg-1", "linux", 10, 10),
				BuildTask("tg-1", "linux", 10, 10),
				BuildTask("tg-2", "linux", 10, 10)})
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Expect(lrpAuctions).To(Equal([]auctiontypes.LRPAuction{
				BuildLRPAuction("pg-1", 1, "linux", 10, 10, clock.Now()),
				BuildLRPAuction("pg-2", 2, "linux", 10, 10, clock.Now()),
			}))

			Expect(taskAuctions).To(Equal([]auctiontypes.TaskAuction{
				BuildTaskAuction(
					BuildTask("tg-1", "linux", 10, 10),
					clock.Now(),
				),
				BuildTaskAuction(
					BuildTask("tg-2", "linux", 10, 10),
					clock.Now(),
				),
			}))

		})

		It("should clear out its cache, so a subsequent call shouldn't fetch anything", func() {
			batch.DedupeAndDrain()
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Expect(lrpAuctions).To(BeEmpty())
			Expect(taskAuctions).To(BeEmpty())
		})

		It("should no longer have work after draining", func() {
			batch.DedupeAndDrain()
			Expect(batch.HasWork).NotTo(Receive())
		})

		It("should not hang forever if the work channel was already drained", func() {
			Expect(batch.HasWork).To(Receive())
			batch.DedupeAndDrain()
			Expect(batch.HasWork).NotTo(Receive())
		})
	})
})
