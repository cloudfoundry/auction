package repnatsserver

import (
	"encoding/json"
	"os"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

type RepNatsServer struct {
	repGuid string
	rep     *auctionrep.AuctionRep
	client  yagnats.NATSClient
	logger  *gosteno.Logger
}

func New(client yagnats.NATSClient, rep *auctionrep.AuctionRep, logger *gosteno.Logger) *RepNatsServer {
	return &RepNatsServer{
		repGuid: rep.Guid(),
		rep:     rep,
		client:  client,
		logger:  logger,
	}
}

func (s *RepNatsServer) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
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

func (s *RepNatsServer) start(subjects nats.Subjects) {
	s.client.Subscribe(subjects.TotalResources, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.total-resources.handling")
		jresources, _ := json.Marshal(s.rep.TotalResources())
		s.client.Publish(msg.ReplyTo, jresources)
	})

	s.client.Subscribe(subjects.BidForStartAuction, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.bid.handling")
		var inst auctiontypes.StartAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.bid.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.StartAuctionBid{
			Rep: s.repGuid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		bid, err := s.rep.BidForStartAuction(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Bid = bid
	})

	s.client.Subscribe(subjects.BidForStopAuction, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.stop-bid.handling")
		var stopAuctionInfo auctiontypes.StopAuctionInfo

		err := json.Unmarshal(msg.Payload, &stopAuctionInfo)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.stop-bid.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.StopAuctionBid{
			Rep: s.repGuid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		bid, instanceGuids, err := s.rep.BidForStopAuction(stopAuctionInfo)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Bid = bid
		response.InstanceGuids = instanceGuids
	})

	s.client.Subscribe(subjects.RebidThenTentativelyReserve, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.bid-then-tentatively-reserve.handling")
		var inst auctiontypes.StartAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.bid-then-tentatively-reserve.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.StartAuctionBid{
			Rep: s.repGuid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		bid, err := s.rep.RebidThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Bid = bid
	})

	s.client.Subscribe(subjects.ReleaseReservation, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.release-reservation.handling")
		var inst auctiontypes.StartAuctionInfo

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.release-reservation.failed-to-unmarshal-auction-info")
			return
		}

		s.rep.ReleaseReservation(inst) //need to handle error

		responsePayload = successResponse
	})

	s.client.Subscribe(subjects.Run, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.run.handling")
		var inst models.LRPStartAuction

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.run.failed-to-unmarshal-auction-info")
			return
		}

		s.rep.Run(inst) //need to handle error

		responsePayload = successResponse
	})

	s.client.Subscribe(subjects.Stop, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"rep-guid": s.repGuid,
		}, "rep-nats-server.stop.handling")
		var instanceGuid string

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &instanceGuid)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error":    err.Error(),
				"rep-guid": s.repGuid,
			}, "rep-nats-server.stop.failed-to-unmarshal-auction-info")
			return
		}

		s.rep.Stop(instanceGuid) //need to handle error

		responsePayload = successResponse
	})

	//simulation only

	s.client.Subscribe(subjects.Reset, func(msg *yagnats.Message) {
		s.rep.Reset()
		s.client.Publish(msg.ReplyTo, successResponse)
	})

	s.client.Subscribe(subjects.SetSimulatedInstances, func(msg *yagnats.Message) {
		var instances []auctiontypes.SimulatedInstance

		err := json.Unmarshal(msg.Payload, &instances)
		if err != nil {
			s.client.Publish(msg.ReplyTo, errorResponse)
		}

		s.rep.SetSimulatedInstances(instances)
		s.client.Publish(msg.ReplyTo, successResponse)
	})

	s.client.Subscribe(subjects.SimulatedInstances, func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(s.rep.SimulatedInstances())
		s.client.Publish(msg.ReplyTo, jinstances)
	})
}

func (s *RepNatsServer) stop(subjects nats.Subjects) {
	for _, topic := range subjects.Slice() {
		s.client.UnsubscribeAll(topic)
	}
}
