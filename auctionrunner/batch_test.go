package auctionrunner_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch", func() {
	var lrpStart models.LRPStartRequest
	var task models.Task
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
	})

	It("should start off empty", func() {
		Ω(batch.HasWork).ShouldNot(Receive())
		starts, tasks := batch.DedupeAndDrain()
		Ω(starts).Should(BeEmpty())
		Ω(tasks).Should(BeEmpty())
	})

	Describe("adding work", func() {
		Context("when adding start auctions", func() {
			BeforeEach(func() {
				lrpStart = BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10)
				batch.AddLRPStarts([]models.LRPStartRequest{lrpStart})
			})

			It("makes the start auction available when drained", func() {
				lrpAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpAuctions).Should(ConsistOf(BuildLRPAuctions(lrpStart, timeProvider.Now())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("when adding tasks", func() {
			BeforeEach(func() {
				task = BuildTask("tg-1", "lucid64", 10, 10)
				batch.AddTasks([]models.Task{task})
			})

			It("makes the stop auction available when drained", func() {
				_, taskAuctions := batch.DedupeAndDrain()
				Ω(taskAuctions).Should(ConsistOf(BuildTaskAuction(task, timeProvider.Now())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})
	})

	Describe("resubmitting work", func() {
		Context("resubmitting starts", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				lrpStartAuction1 := BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10)
				startAuction1 := BuildLRPAuctions(lrpStartAuction1, timeProvider.Now())

				lrpStartAuction2 := BuildLRPStartRequest("pg-2", []uint{1}, "lucid64", 10, 10)
				startAuction2 := BuildLRPAuctions(lrpStartAuction2, timeProvider.Now())

				batch.AddLRPStarts([]models.LRPStartRequest{lrpStartAuction1, lrpStartAuction2})

				lrpAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpAuctions).Should(Equal(append(startAuction1, startAuction2...)))

				batch.AddLRPStarts([]models.LRPStartRequest{lrpStartAuction1, lrpStartAuction2})
				batch.ResubmitStartAuctions(startAuction2)

				lrpAuctions, _ = batch.DedupeAndDrain()
				Ω(lrpAuctions).Should(Equal(append(startAuction2, startAuction1...)))
			})

			It("should have work", func() {
				lrpStartAuction := BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10)
				startAuction := BuildLRPAuctions(lrpStartAuction, timeProvider.Now())
				batch.ResubmitStartAuctions(startAuction)

				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("resubmitting tasks", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				task1 := BuildTask("tg-1", "lucid64", 10, 10)
				taskAuction1 := BuildTaskAuction(task1, timeProvider.Now())

				task2 := BuildTask("tg-2", "lucid64", 10, 10)
				taskAuction2 := BuildTaskAuction(task2, timeProvider.Now())

				batch.AddTasks([]models.Task{task1, task2})

				_, taskAuctions := batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction1, taskAuction2}))

				batch.AddTasks([]models.Task{task1, task2})
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction2})

				_, taskAuctions = batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction2, taskAuction1}))
			})

			It("should have work", func() {
				task := BuildTask("tg-1", "lucid64", 10, 10)
				taskAuction := BuildTaskAuction(task, timeProvider.Now())
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})
	})

	Describe("DedupeAndDrain", func() {
		BeforeEach(func() {
			batch.AddLRPStarts([]models.LRPStartRequest{
				BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10),
				BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10),
				BuildLRPStartRequest("pg-2", []uint{2}, "lucid64", 10, 10),
			})

			batch.AddTasks([]models.Task{
				BuildTask("tg-1", "lucid64", 10, 10),
				BuildTask("tg-1", "lucid64", 10, 10),
				BuildTask("tg-2", "lucid64", 10, 10)})
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpAuctions).Should(Equal([]auctiontypes.LRPAuction{
				BuildLRPAuction("pg-1", 1, "lucid64", 10, 10, timeProvider.Now()),
				BuildLRPAuction("pg-2", 2, "lucid64", 10, 10, timeProvider.Now()),
			}))

			Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{
				BuildTaskAuction(
					BuildTask("tg-1", "lucid64", 10, 10),
					timeProvider.Now(),
				),
				BuildTaskAuction(
					BuildTask("tg-2", "lucid64", 10, 10),
					timeProvider.Now(),
				),
			}))
		})

		It("should clear out its cache, so a subsequent call shouldn't fetch anything", func() {
			batch.DedupeAndDrain()
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpAuctions).Should(BeEmpty())
			Ω(taskAuctions).Should(BeEmpty())
		})

		It("should no longer have work after draining", func() {
			batch.DedupeAndDrain()
			Ω(batch.HasWork).ShouldNot(Receive())
		})

		It("should not hang forever if the work channel was already drained", func() {
			Ω(batch.HasWork).Should(Receive())
			batch.DedupeAndDrain()
			Ω(batch.HasWork).ShouldNot(Receive())
		})
	})
})
