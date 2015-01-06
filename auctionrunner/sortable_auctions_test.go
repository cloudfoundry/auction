package auctionrunner_test

import (
	"sort"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sortable Auctions", func() {
	Describe("LRP Auctions", func() {
		var lrps []auctiontypes.LRPAuction

		JustBeforeEach(func() {
			sort.Sort(auctionrunner.SortableLRPAuctions(lrps))
		})

		Context("when LRP indexes match", func() {
			BeforeEach(func() {
				lrps = []auctiontypes.LRPAuction{
					BuildLRPAuction("pg-6", 0, "lucid64", 10, 10, time.Time{}),
					BuildLRPAuction("pg-7", 0, "lucid64", 20, 10, time.Time{}),
					BuildLRPAuction("pg-8", 0, "lucid64", 30, 10, time.Time{}),
					BuildLRPAuction("pg-9", 0, "lucid64", 40, 10, time.Time{}),
				}
			})

			It("sorts boulders before pebbles", func() {
				Ω(lrps[0].DesiredLRP.ProcessGuid).Should((Equal("pg-9")))
				Ω(lrps[1].DesiredLRP.ProcessGuid).Should((Equal("pg-8")))
				Ω(lrps[2].DesiredLRP.ProcessGuid).Should((Equal("pg-7")))
				Ω(lrps[3].DesiredLRP.ProcessGuid).Should((Equal("pg-6")))
			})
		})

		Context("when LRP indexes differ", func() {
			BeforeEach(func() {
				lrps = make([]auctiontypes.LRPAuction, 5)
				for i := cap(lrps) - 1; i >= 0; i-- {
					lrps[i] = BuildLRPAuction("pg", i, "lucid64", 40+i, 40+i, time.Time{})
				}
			})

			It("sorts by index", func() {
				for i := 0; i < len(lrps); i++ {
					Ω(lrps[i].Index).Should(Equal(i))
				}
			})
		})
	})

	Describe("Task Auctions", func() {
		var tasks []auctiontypes.TaskAuction

		BeforeEach(func() {
			tasks = []auctiontypes.TaskAuction{
				BuildTaskAuction(BuildTask("tg-6", "lucid64", 10, 10), time.Time{}),
				BuildTaskAuction(BuildTask("tg-7", "lucid64", 20, 10), time.Time{}),
				BuildTaskAuction(BuildTask("tg-8", "lucid64", 30, 10), time.Time{}),
				BuildTaskAuction(BuildTask("tg-9", "lucid64", 40, 10), time.Time{}),
			}

			sort.Sort(auctionrunner.SortableTaskAuctions(tasks))
		})

		It("sorts boulders before pebbles", func() {
			Ω(tasks[0].Task.TaskGuid).Should((Equal("tg-9")))
			Ω(tasks[1].Task.TaskGuid).Should((Equal("tg-8")))
			Ω(tasks[2].Task.TaskGuid).Should((Equal("tg-7")))
			Ω(tasks[3].Task.TaskGuid).Should((Equal("tg-6")))
		})

	})
})
