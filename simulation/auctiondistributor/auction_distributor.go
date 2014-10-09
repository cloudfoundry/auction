package auctiondistributor

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type AuctionDistributor interface {
	HoldStartAuctions(numAuctioneers int, startAuctions []models.LRPStartAuction, repAddresses []auctiontypes.RepAddress, rules auctiontypes.StartAuctionRules) []auctiontypes.StartAuctionResult
	HoldStopAuctions(numAuctioneers int, stopAuctions []models.LRPStopAuction, repAddresses []auctiontypes.RepAddress) []auctiontypes.StopAuctionResult
}

func buildStartAuctionRequests(startAuctions []models.LRPStartAuction, repAddresses []auctiontypes.RepAddress, rules auctiontypes.StartAuctionRules) []auctiontypes.StartAuctionRequest {
	requests := []auctiontypes.StartAuctionRequest{}
	for _, startAuction := range startAuctions {
		requests = append(requests, auctiontypes.StartAuctionRequest{
			LRPStartAuction: startAuction,
			RepAddresses:    repAddresses,
			Rules:           rules,
		})
	}
	return requests
}

func buildStopAuctionRequests(stopAuctions []models.LRPStopAuction, repAddresses []auctiontypes.RepAddress) []auctiontypes.StopAuctionRequest {
	requests := []auctiontypes.StopAuctionRequest{}
	for _, stopAuction := range stopAuctions {
		requests = append(requests, auctiontypes.StopAuctionRequest{
			LRPStopAuction: stopAuction,
			RepAddresses:   repAddresses,
		})
	}
	return requests
}
