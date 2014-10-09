package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

type bidForStopAuction struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *bidForStopAuction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("stop-auction-bid")
	logger.Info("handling")

	var stopAuctionInfo auctiontypes.StopAuctionInfo
	if !decodeJSON(w, r, &stopAuctionInfo, logger) {
		return
	}

	logger = logger.WithData(lager.Data{
		"process-guid": stopAuctionInfo.ProcessGuid,
		"index":        stopAuctionInfo.Index,
	})

	response := auctiontypes.StopAuctionBid{
		Rep: h.rep.Guid(),
	}

	var status int
	bid, instanceGuids, err := h.rep.BidForStopAuction(stopAuctionInfo)
	if err != nil {
		status = http.StatusForbidden
		response.Error = err.Error()
		logger.Info("not-bidding", lager.Data{"reason": err.Error()})
	} else {
		status = http.StatusOK
		response.Bid = bid
		response.InstanceGuids = instanceGuids
		logger.Info("bidding", lager.Data{"bid": bid, "instance-guids": instanceGuids})
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)

	logger.Info("success")
}
