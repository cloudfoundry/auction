package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

type perform struct {
	rep    auctiontypes.CellRep
	logger lager.Logger
}

func (h *perform) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("auction-perform-work")
	logger.Info("handling")

	var work auctiontypes.Work
	err := json.NewDecoder(r.Body).Decode(&work)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error("failed-to-unmarshal", err)
		return
	}

	failedWork, err := h.rep.Perform(work)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("failed-to-perform-work", err)
		return
	}

	json.NewEncoder(w).Encode(failedWork)
	logger.Info("success")
}
