package auction_http_handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

type reset struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *reset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("sim-reset")
	logger.Info("handling")

	simRep, ok := h.rep.(auctiontypes.SimulationAuctionRep)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("not-a-simulation-rep", nil)
		return
	}

	err := simRep.Reset()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("failed-to-reset", err)
		return
	}
	logger.Info("success")
}
