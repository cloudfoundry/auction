package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type rebidThenReserve struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *rebidThenReserve) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("rebid-then-reserve")
	logger.Info("handling")

	var startAuction models.LRPStartAuction
	if !decodeJSON(w, r, &startAuction, logger) {
		return
	}

	logger = logger.WithData(lagerDataForStartAuction(startAuction))

	response := auctiontypes.StartAuctionBid{
		Rep: h.rep.Guid(),
	}

	var status int
	bid, err := h.rep.RebidThenTentativelyReserve(startAuction)
	if err != nil {
		status = http.StatusForbidden
		response.Error = err.Error()
		logger.Info("not-bidding", lager.Data{"reason": err.Error()})
	} else {
		status = http.StatusOK
		response.Bid = bid
		logger.Info("bidding", lager.Data{"bid": bid})
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)

	logger.Info("success")
}
