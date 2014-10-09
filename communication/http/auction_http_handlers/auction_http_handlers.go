package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func New(rep auctiontypes.AuctionRep, logger lager.Logger) rata.Handlers {
	handlers := rata.Handlers{
		routes.BidForStartAuction:          &bidForStartAuction{rep: rep, logger: logger},
		routes.BidForStopAuction:           &bidForStopAuction{rep: rep, logger: logger},
		routes.RebidThenTentativelyReserve: &rebidThenReserve{rep: rep, logger: logger},
		routes.ReleaseReservation:          &releaseReservation{rep: rep, logger: logger},
		routes.Run:                         &run{rep: rep, logger: logger},
		routes.Stop:                        &stop{rep: rep, logger: logger},

		routes.Sim_TotalResources:        &sim_TotalResources{rep: rep, logger: logger},
		routes.Sim_Reset:                 &sim_Reset{rep: rep, logger: logger},
		routes.Sim_SimulatedInstances:    &sim_SimulatedInstances{rep: rep, logger: logger},
		routes.Sim_SetSimulatedInstances: &sim_SetSimulatedInstances{rep: rep, logger: logger},
	}

	return handlers
}

func lagerDataForStartAuctionInfo(startAuctionInfo auctiontypes.StartAuctionInfo) lager.Data {
	return lager.Data{
		"process-guid":  startAuctionInfo.ProcessGuid,
		"instance-guid": startAuctionInfo.InstanceGuid,
		"disk-mb":       startAuctionInfo.DiskMB,
		"memory-mb":     startAuctionInfo.MemoryMB,
		"index":         startAuctionInfo.Index,
	}
}

func lagerDataForStartAuction(startAuction models.LRPStartAuction) lager.Data {
	return lager.Data{
		"process-guid":  startAuction.DesiredLRP.ProcessGuid,
		"instance-guid": startAuction.InstanceGuid,
		"disk-mb":       startAuction.DesiredLRP.DiskMB,
		"memory-mb":     startAuction.DesiredLRP.MemoryMB,
		"index":         startAuction.Index,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, into interface{}, logger lager.Logger) bool {
	err := json.NewDecoder(r.Body).Decode(into)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json: " + err.Error()))
		logger.Error("invalid-json", err)
		return false
	}

	return true
}
