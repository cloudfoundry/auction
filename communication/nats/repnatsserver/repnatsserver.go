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
	guid   string
	rep    *auctionrep.AuctionRep
	client yagnats.NATSClient
	logger *gosteno.Logger
}

func New(client yagnats.NATSClient, rep *auctionrep.AuctionRep, logger *gosteno.Logger) *RepNatsServer {
	return &RepNatsServer{
		guid:   rep.Guid(),
		rep:    rep,
		client: client,
		logger: logger,
	}
}

func (s *RepNatsServer) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	subjects := nats.NewSubjects(s.guid)

	s.start(subjects)
	s.logger.Infod(map[string]interface{}{
		"guid": s.guid,
	}, "rep-nats-server.listening")
	close(ready)

	<-sigChan
	s.stop(subjects)
	return nil
}

func (s *RepNatsServer) start(subjects nats.Subjects) {
	s.client.Subscribe(subjects.TotalResources, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.total-resources.handling")
		jresources, _ := json.Marshal(s.rep.TotalResources())
		s.client.Publish(msg.ReplyTo, jresources)
	})

	s.client.Subscribe(subjects.Score, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.score.handling")
		var inst auctiontypes.LRPStartAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
			}, "rep-nats-server.score.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.ScoreResult{
			Rep: s.guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		score, err := s.rep.Score(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	s.client.Subscribe(subjects.StopScore, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.stop-score.handling")
		var stopAuctionInfo auctiontypes.LRPStopAuctionInfo

		err := json.Unmarshal(msg.Payload, &stopAuctionInfo)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
			}, "rep-nats-server.stop-score.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.StopScoreResult{
			Rep: s.guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		score, instanceGuids, err := s.rep.StopScore(stopAuctionInfo)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
		response.InstanceGuids = instanceGuids
	})

	s.client.Subscribe(subjects.ScoreThenTentativelyReserve, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.score-then-tentatively-reserve.handling")
		var inst auctiontypes.LRPStartAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
			}, "rep-nats-server.score-then-tentatively-reserve.failed-to-unmarshal-auction-info")
			return
		}

		response := auctiontypes.ScoreResult{
			Rep: s.guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			s.client.Publish(msg.ReplyTo, payload)
		}()

		score, err := s.rep.ScoreThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	s.client.Subscribe(subjects.ReleaseReservation, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.release-reservation.handling")
		var inst auctiontypes.LRPStartAuctionInfo

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
			}, "rep-nats-server.release-reservation.failed-to-unmarshal-auction-info")
			return
		}

		s.rep.ReleaseReservation(inst) //need to handle error

		responsePayload = successResponse
	})

	s.client.Subscribe(subjects.Run, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.run.handling")
		var inst models.LRPStartAuction

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
			}, "rep-nats-server.run.failed-to-unmarshal-auction-info")
			return
		}

		s.rep.Run(inst) //need to handle error

		responsePayload = successResponse
	})

	s.client.Subscribe(subjects.Stop, func(msg *yagnats.Message) {
		s.logger.Infod(map[string]interface{}{
			"guid": s.guid,
		}, "rep-nats-server.stop.handling")
		var instanceGuid string

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &instanceGuid)
		if err != nil {
			s.logger.Errord(map[string]interface{}{
				"error": err.Error(),
				"guid":  s.guid,
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
