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

func accumulateZonesByInstances(zones map[string]Zone, lrpAuction auctiontypes.LRPAuction) []lrpByZone {
	lrpZones := []lrpByZone{}

	for _, zone := range zones {
		instances := 0
		for _, cell := range zone {
			for _, lrp := range cell.state.LRPs {
				if lrp.ProcessGuid == lrpAuction.DesiredLRP.ProcessGuid {
					instances++
				}
			}
		}
		lrpZones = append(lrpZones, lrpByZone{zone, instances})
	}
	return lrpZones
}

func sortZonesByInstances(zones []lrpByZone) []lrpByZone {
	sorter := zoneSorterByInstances{zones: zones}
	sort.Sort(sorter)
	return sorter.zones
}

func filterZonesByRootFS(zones []lrpByZone, rootFS string) []lrpByZone {
	filteredZones := []lrpByZone{}

	for _, lrpZone := range zones {
		cells := lrpZone.zone.FilterCells(rootFS)
		if len(cells) > 0 {
			filteredZone := lrpByZone{
				zone:      Zone(cells),
				instances: lrpZone.instances,
			}
			filteredZones = append(filteredZones, filteredZone)
		}
	}

	return filteredZones
}
