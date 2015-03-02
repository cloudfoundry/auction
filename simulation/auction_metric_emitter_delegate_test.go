package simulation_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type auctionMetricEmitterDelegate struct{}

func NewAuctionMetricEmitterDelegate() auctionMetricEmitterDelegate {
	return auctionMetricEmitterDelegate{}
}

func (_ auctionMetricEmitterDelegate) FetchStatesCompleted(_ time.Duration) {}

func (_ auctionMetricEmitterDelegate) AuctionCompleted(_ auctiontypes.AuctionResults) {}
