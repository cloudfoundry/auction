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
	var lrpStopAuction models.LRPStopAuction
	var task models.Task
	var batch *Batch
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		batch = NewBatch(timeProvider)
	})

	It("should start off empty", func() {
		Ω(batch.HasWork).ShouldNot(Receive())
		starts, stops, tasks := batch.DedupeAndDrain()
		Ω(starts).Should(BeEmpty())
		Ω(stops).Should(BeEmpty())
		Ω(tasks).Should(BeEmpty())
	})

	Describe("adding work", func() {
		Context("when adding start auctions", func() {
			BeforeEach(func() {
				lrpStartAuction = BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10)
				batch.AddLRPStartAuction(lrpStartAuction)
			})

			It("makes the start auction available when drained", func() {
				lrpStartAuctions, _, _ := batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(ConsistOf(BuildStartAuction(lrpStartAuction, timeProvider.Now())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("when adding stop auctions", func() {
			BeforeEach(func() {
				lrpStopAuction = BuildLRPStopAuction("pg-1", 1)
				batch.AddLRPStopAuction(lrpStopAuction)
			})

			It("makes the stop auction available when drained", func() {
				_, lrpStopAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpStopAuctions).Should(ConsistOf(BuildStopAuction(lrpStopAuction, timeProvider.Now())))
			})

			It("should have work", func() {
				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("when adding tasks", func() {
			BeforeEach(func() {
				task = BuildTask("tg-1")
				batch.AddTask(task)
			})

			It("makes the stop auction available when drained", func() {
				_, _, taskAuctions := batch.DedupeAndDrain()
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
				lrpStartAuction1 := BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10)
				startAuction1 := BuildStartAuction(lrpStartAuction1, timeProvider.Now())

				lrpStartAuction2 := BuildLRPStartAuction("pg-2", "ig-2", 1, "lucid64", 10, 10)
				startAuction2 := BuildStartAuction(lrpStartAuction2, timeProvider.Now())

				batch.AddLRPStartAuction(lrpStartAuction1)
				batch.AddLRPStartAuction(lrpStartAuction2)

				lrpStartAuctions, _, _ := batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{startAuction1, startAuction2}))

				batch.AddLRPStartAuction(lrpStartAuction1)
				batch.AddLRPStartAuction(lrpStartAuction2)
				batch.ResubmitStartAuctions([]auctiontypes.LRPStartAuction{startAuction2})

				lrpStartAuctions, _, _ = batch.DedupeAndDrain()
				Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{startAuction2, startAuction1}))
			})

			It("should have work", func() {
				lrpStartAuction := BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10)
				startAuction := BuildStartAuction(lrpStartAuction, timeProvider.Now())
				batch.ResubmitStartAuctions([]auctiontypes.LRPStartAuction{startAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("resubmitting stops", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				lrpStopAuction1 := BuildLRPStopAuction("pg-1", 1)
				stopAuction1 := BuildStopAuction(lrpStopAuction1, timeProvider.Now())

				lrpStopAuction2 := BuildLRPStopAuction("pg-2", 1)
				stopAuction2 := BuildStopAuction(lrpStopAuction2, timeProvider.Now())

				batch.AddLRPStopAuction(lrpStopAuction1)
				batch.AddLRPStopAuction(lrpStopAuction2)

				_, lrpStopAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpStopAuctions).Should(Equal([]auctiontypes.LRPStopAuction{stopAuction1, stopAuction2}))

				batch.AddLRPStopAuction(lrpStopAuction1)
				batch.AddLRPStopAuction(lrpStopAuction2)
				batch.ResubmitStopAuctions([]auctiontypes.LRPStopAuction{stopAuction2})

				_, lrpStopAuctions, _ = batch.DedupeAndDrain()
				Ω(lrpStopAuctions).Should(Equal([]auctiontypes.LRPStopAuction{stopAuction2, stopAuction1}))
			})

			It("should have work", func() {
				lrpStopAuction := BuildLRPStopAuction("pg-1", 1)
				stopAuction := BuildStopAuction(lrpStopAuction, timeProvider.Now())
				batch.ResubmitStopAuctions([]auctiontypes.LRPStopAuction{stopAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("resubmitting tasks", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				task1 := BuildTask("tg-1")
				taskAuction1 := BuildTaskAuction(task1, timeProvider.Now())

				task2 := BuildTask("tg-2")
				taskAuction2 := BuildTaskAuction(task2, timeProvider.Now())

				batch.AddTask(task1)
				batch.AddTask(task2)

				_, _, taskAuctions := batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction1, taskAuction2}))

				batch.AddTask(task1)
				batch.AddTask(task2)
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction2})

				_, _, taskAuctions = batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction2, taskAuction1}))
			})

			It("should have work", func() {
				task := BuildTask("tg-1")
				taskAuction := BuildTaskAuction(task, timeProvider.Now())
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})
	})

	Describe("DedupeAndDrain", func() {
		BeforeEach(func() {
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10))
			batch.AddLRPStartAuction(BuildLRPStartAuction("pg-2", "ig-2", 2, "lucid64", 10, 10))

			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-1", 1))
			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-1", 1))
			batch.AddLRPStopAuction(BuildLRPStopAuction("pg-2", 3))

			batch.AddTask(BuildTask("tg-1"))
			batch.AddTask(BuildTask("tg-1"))
			batch.AddTask(BuildTask("tg-2"))
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			lrpStartAuctions, lrpStopAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpStartAuctions).Should(Equal([]auctiontypes.LRPStartAuction{
				BuildStartAuction(
					BuildLRPStartAuction("pg-1", "ig-1", 1, "lucid64", 10, 10),
					timeProvider.Now(),
				),
				BuildStartAuction(
					BuildLRPStartAuction("pg-2", "ig-2", 2, "lucid64", 10, 10),
					timeProvider.Now(),
				),
			}))

			Ω(lrpStopAuctions).Should(Equal([]auctiontypes.LRPStopAuction{
				BuildStopAuction(
					BuildLRPStopAuction("pg-1", 1),
					timeProvider.Now(),
				),
				BuildStopAuction(
					BuildLRPStopAuction("pg-2", 3),
					timeProvider.Now(),
				),
			}))

			Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{
				BuildTaskAuction(
					BuildTask("tg-1"),
					timeProvider.Now(),
				),
				BuildTaskAuction(
					BuildTask("tg-2"),
					timeProvider.Now(),
				),
			}))
		})

		It("should clear out its cache, so a subsequent call shouldn't fetch anything", func() {
			batch.DedupeAndDrain()
			lrpStartAuctions, lrpStopAuctions, taskAuctions := batch.DedupeAndDrain()
			Ω(lrpStartAuctions).Should(BeEmpty())
			Ω(lrpStopAuctions).Should(BeEmpty())
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
