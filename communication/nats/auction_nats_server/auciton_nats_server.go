package auction_nats_server

import (
	"encoding/json"
	"os"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/auction/communication/nats/nats_muxer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

type AuctionNATSServer struct {
	repGuid string
	rep     *auctionrep.AuctionRep
	client  yagnats.NATSClient
	logger  *gosteno.Logger
}

func New(client yagnats.NATSClient, rep *auctionrep.AuctionRep, logger *gosteno.Logger) *AuctionNATSServer {
	return &AuctionNATSServer{
		repGuid: rep.Guid(),
		rep:     rep,
		client:  client,
		logger:  logger,
	}
}

func (s *AuctionNATSServer) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	subjects := nats.NewSubjects(s.repGuid)

	s.start(subjects)
	s.logger.Infod(map[string]interface{}{
		"rep-guid": s.repGuid,
	}, "rep-nats-server.listening")
	close(ready)

	<-sigChan
	s.stop(subjects)
	return nil
}

func (s *AuctionNATSServer) start(subjects nats.Subjects) {
	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.TotalResources, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.total-resources.handling")

		out, _ := json.Marshal(s.rep.TotalResources())
		return out
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.BidForStartAuction, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.bid.handling")
		var inst auctiontypes.StartAuctionInfo

		err := json.Unmarshal(payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.bid.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		response := auctiontypes.StartAuctionBid{
			Rep: s.repGuid,
		}

		bid, err := s.rep.BidForStartAuction(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Bid = bid
		}

		out, _ := json.Marshal(response)
		return out
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.BidForStopAuction, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.stop-bid.handling")
		var stopAuctionInfo auctiontypes.StopAuctionInfo

		err := json.Unmarshal(payload, &stopAuctionInfo)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.stop-bid.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		response := auctiontypes.StopAuctionBid{
			Rep: s.repGuid,
		}

		bid, instanceGuids, err := s.rep.BidForStopAuction(stopAuctionInfo)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Bid = bid
			response.InstanceGuids = instanceGuids
		}

		out, _ := json.Marshal(response)
		return out
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.RebidThenTentativelyReserve, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.bid-then-tentatively-reserve.handling")
		var inst auctiontypes.StartAuctionInfo

		err := json.Unmarshal(payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.bid-then-tentatively-reserve.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		response := auctiontypes.StartAuctionBid{
			Rep: s.repGuid,
		}

		bid, err := s.rep.RebidThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Bid = bid
		}

		out, _ := json.Marshal(response)
		return out
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.ReleaseReservation, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.release-reservation.handling")
		var inst auctiontypes.StartAuctionInfo

		err := json.Unmarshal(payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.release-reservation.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		s.rep.ReleaseReservation(inst) //need to handle error

		return successResponse
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.Run, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.run.handling")
		var inst models.LRPStartAuction

		err := json.Unmarshal(payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.run.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		s.rep.Run(inst) //need to handle error

		return successResponse
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.Stop, func(payload []byte) []byte {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.stop.handling")
		var stopInstance models.StopLRPInstance

		err := json.Unmarshal(payload, &stopInstance)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.stop.failed-to-unmarshal-auction-info")
			return errorResponse
		}

		s.rep.Stop(stopInstance) //need to handle error

		return successResponse
	})

	//simulation only

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.Reset, func(payload []byte) []byte {
		s.rep.Reset()
		return successResponse
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.SetSimulatedInstances, func(payload []byte) []byte {
		var instances []auctiontypes.SimulatedInstance

		err := json.Unmarshal(payload, &instances)
		if err != nil {
			return errorResponse
		}

		s.rep.SetSimulatedInstances(instances)
		return successResponse
	})

	nats_muxer.HandleMuxedNATSRequest(s.client, subjects.SimulatedInstances, func(payload []byte) []byte {
		jinstances, _ := json.Marshal(s.rep.SimulatedInstances())
		return jinstances
	})
}

func (s *AuctionNATSServer) stop(subjects nats.Subjects) {
	for _, topic := range subjects.Slice() {
		s.client.UnsubscribeAll(topic)
	}
}
