package auctionrunner_test

import (
	"sort"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SortableAuction", func() {
	var lrps []auctiontypes.LRPAuction

	JustBeforeEach(func() {
		sort.Sort(auctionrunner.SortableAuctions(lrps))
	})

	Context("sorts boulders before pebbles", func() {
		BeforeEach(func() {
			lrps = []auctiontypes.LRPAuction{
				BuildLRPAuction("pg-6", 0, "lucid64", 10, 10, time.Time{}),
				BuildLRPAuction("pg-7", 0, "lucid64", 20, 10, time.Time{}),
				BuildLRPAuction("pg-8", 0, "lucid64", 30, 10, time.Time{}),
				BuildLRPAuction("pg-9", 0, "lucid64", 40, 10, time.Time{}),
			}
		})

		It("has the correct order", func() {
			Ω(lrps[0].DesiredLRP.ProcessGuid).Should((Equal("pg-9")))
			Ω(lrps[1].DesiredLRP.ProcessGuid).Should((Equal("pg-8")))
			Ω(lrps[2].DesiredLRP.ProcessGuid).Should((Equal("pg-7")))
			Ω(lrps[3].DesiredLRP.ProcessGuid).Should((Equal("pg-6")))
		})
	})
})
