package auction_http_handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type run struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *run) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("run")
	logger.Info("handling")

	var lrpStartAuction models.LRPStartAuction
	if !decodeJSON(w, r, &lrpStartAuction, logger) {
		return
	}

	logger = logger.WithData(lager.Data{
		"process-guid":  lrpStartAuction.DesiredLRP.ProcessGuid,
		"instance-guid": lrpStartAuction.InstanceGuid,
		"index":         lrpStartAuction.Index,
	})

	err := h.rep.Run(lrpStartAuction)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(err.Error()))
		logger.Error("failed", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	logger.Info("success")
}
