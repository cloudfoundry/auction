package auction_http_handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type releaseReservation struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *releaseReservation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("release-reservation")
	logger.Info("handling")

	var startAuction models.LRPStartAuction
	if !decodeJSON(w, r, &startAuction, logger) {
		return
	}

	logger = logger.WithData(lagerDataForStartAuction(startAuction))

	err := h.rep.ReleaseReservation(startAuction)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(err.Error()))
		logger.Error("failed", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	logger.Info("success")
}
