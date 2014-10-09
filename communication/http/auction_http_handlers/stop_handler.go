package auction_http_handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type stop struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *stop) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("stop")
	logger.Info("handling")

	var stopLRPInstance models.StopLRPInstance
	if !decodeJSON(w, r, &stopLRPInstance, logger) {
		return
	}

	logger = logger.WithData(lager.Data{
		"process-guid":  stopLRPInstance.ProcessGuid,
		"instance-guid": stopLRPInstance.InstanceGuid,
		"index":         stopLRPInstance.Index,
	})

	err := h.rep.Stop(stopLRPInstance)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(err.Error()))
		logger.Error("failed", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	logger.Info("success")
}
