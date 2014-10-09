package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

///

type sim_TotalResources struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *sim_TotalResources) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("sim-total-resources")
	logger.Info("handling")

	simRep, ok := h.rep.(auctiontypes.SimulationAuctionRep)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("invalid auction rep"))
		logger.Error("invalid-auction-rep", nil)
		return
	}

	resources := simRep.TotalResources()
	logger.Info("success")
	json.NewEncoder(w).Encode(resources)
}

///

type sim_Reset struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *sim_Reset) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("sim-reset")
	logger.Info("handling")

	simRep, ok := h.rep.(auctiontypes.SimulationAuctionRep)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("invalid auction rep"))
		logger.Error("invalid-auction-rep", nil)
		return
	}

	simRep.Reset()

	logger.Info("success")
}

///

type sim_SimulatedInstances struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *sim_SimulatedInstances) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("sim-simulated-instances")
	logger.Info("handling")

	simRep, ok := h.rep.(auctiontypes.SimulationAuctionRep)

	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("invalid auction rep"))
		logger.Error("invalid-auction-rep", nil)
		return
	}

	simulatedInstances := simRep.SimulatedInstances()
	json.NewEncoder(w).Encode(simulatedInstances)

	logger.Info("success")
}

///

type sim_SetSimulatedInstances struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *sim_SetSimulatedInstances) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("sim-set-simulated-instances")
	logger.Info("handling")

	simRep, ok := h.rep.(auctiontypes.SimulationAuctionRep)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("invalid-auction-rep", nil)
		return
	}

	var simulatedInstances []auctiontypes.SimulatedInstance
	err := json.NewDecoder(r.Body).Decode(&simulatedInstances)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json: " + err.Error()))
		logger.Error("invalid-json", err)
		return
	}

	simRep.SetSimulatedInstances(simulatedInstances)

	logger.Info("success")
}
