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
	var lrpStartAuction models.LRPStartAuction
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
				lrpStartAuction = BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10)
				batch.AddLRPStartAuction(lrpStartAuction)
			})

			It("makes the start auction available when drained", func() {
				lrpStartAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(ConsistOf(BuildStartAuction(lrpStartAuction, timeProvider.Now())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("when adding tasks", func() {
			BeforeEach(func() {
				task = BuildTask("tg-1", "lucid64", 10, 10)
				batch.AddTask(task)
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
				lrpStartAuction1 := BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10)
				startAuction1 := BuildStartAuction(lrpStartAuction1, timeProvider.Now())

				lrpStartAuction2 := BuildLRPStartAuction("pg-2", 1, "lucid64", 10, 10)
				startAuction2 := BuildStartAuction(lrpStartAuction2, timeProvider.Now())

				batch.AddLRPStartAuction(lrpStartAuction1)
				batch.AddLRPStartAuction(lrpStartAuction2)

				lrpStartAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{startAuction1, startAuction2}))

				batch.AddLRPStartAuction(lrpStartAuction1)
				batch.AddLRPStartAuction(lrpStartAuction2)
				batch.ResubmitStartAuctions([]auctiontypes.LRPStartAuction{startAuction2})

				lrpStartAuctions, _ = batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{startAuction2, startAuction1}))
			})

			It("should have work", func() {
				lrpStartAuction := BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10)
				startAuction := BuildStartAuction(lrpStartAuction, timeProvider.Now())
				batch.ResubmitStartAuctions([]auctiontypes.LRPStartAuction{startAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("resubmitting tasks", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				task1 := BuildTask("tg-1", "lucid64", 10, 10)
				taskAuction1 := BuildTaskAuction(task1, timeProvider.Now())

				task2 := BuildTask("tg-2", "lucid64", 10, 10)
				taskAuction2 := BuildTaskAuction(task2, timeProvider.Now())

				batch.AddTask(task1)
				batch.AddTask(task2)

				_, taskAuctions := batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction1, taskAuction2}))

				batch.AddTask(task1)
				batch.AddTask(task2)
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
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-2", 2, "lucid64", 10, 10))

			batch.AddTask(BuildTask("tg-1", "lucid64", 10, 10))
			batch.AddTask(BuildTask("tg-1", "lucid64", 10, 10))
			batch.AddTask(BuildTask("tg-2", "lucid64", 10, 10))
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			lrpStartAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{
				BuildStartAuction(
					BuildLRPStartAuction("pg-1", 1, "lucid64", 10, 10),
					timeProvider.Now(),
				),
				BuildStartAuction(
					BuildLRPStartAuction("pg-2", 2, "lucid64", 10, 10),
					timeProvider.Now(),
				),
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
			lrpStartAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpStartAuctions).Should(BeEmpty())
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
