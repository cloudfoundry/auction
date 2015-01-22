package auctionrunner

import (
	"sort"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type lrpByZone struct {
	zone      Zone
	instances int
}

type zoneSorterByInstances struct {
	zones []lrpByZone
}

func (s zoneSorterByInstances) Len() int           { return len(s.zones) }
func (s zoneSorterByInstances) Swap(i, j int)      { s.zones[i], s.zones[j] = s.zones[j], s.zones[i] }
func (s zoneSorterByInstances) Less(i, j int) bool { return s.zones[i].instances < s.zones[j].instances }

func sortZonesByInstances(zones map[string]Zone, lrpAuction auctiontypes.LRPAuction) []lrpByZone {
	sorter := zoneSorterByInstances{}

	for _, zone := range zones {
		instances := 0
		for _, cell := range zone {
			for _, lrp := range cell.state.LRPs {
				if lrp.ProcessGuid == lrpAuction.DesiredLRP.ProcessGuid {
					instances++
				}
			}
		}
		sorter.zones = append(sorter.zones, lrpByZone{zone, instances})
	}

	sort.Sort(sorter)
	return sorter.zones
}
