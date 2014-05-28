package repnatsserver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/yagnats"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

type RepNatsServer struct {
	guid   string
	rep    *auctionrep.AuctionRep
	client yagnats.NATSClient
}

func New(client yagnats.NATSClient, rep *auctionrep.AuctionRep) *RepNatsServer {
	return &RepNatsServer{
		guid:   rep.Guid(),
		rep:    rep,
		client: client,
	}
}

func (s *RepNatsServer) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	subjects := nats.NewSubjects(s.guid)

	s.start(subjects)
	fmt.Println("rep nats server", s.guid, "listening")
	close(ready)

	<-sigChan
	s.stop(subjects)
	return nil
}

func (s *RepNatsServer) start(subjects nats.Subjects) {
	s.client.Subscribe(subjects.TotalResources, func(msg *yagnats.Message) {
		jresources, _ := json.Marshal(s.rep.TotalResources())
		s.client.Publish(msg.ReplyTo, jresources)
	})

	s.client.Subscribe(subjects.Reset, func(msg *yagnats.Message) {
		s.rep.Reset()
		s.client.Publish(msg.ReplyTo, successResponse)
	})

	s.client.Subscribe(subjects.SetLrpAuctionInfos, func(msg *yagnats.Message) {
		var instances []auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &instances)
		if err != nil {
			s.client.Publish(msg.ReplyTo, errorResponse)
		}

		s.rep.SetLRPAuctionInfos(instances)
		s.client.Publish(msg.ReplyTo, successResponse)
	})

	s.client.Subscribe(subjects.LrpAuctionInfos, func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(s.rep.LRPAuctionInfos())
		s.client.Publish(msg.ReplyTo, jinstances)
	})

	s.client.Subscribe(subjects.Score, func(msg *yagnats.Message) {
		var inst auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
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

	s.client.Subscribe(subjects.ScoreThenTentativelyReserve, func(msg *yagnats.Message) {
		var inst auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
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
		var inst auctiontypes.LRPAuctionInfo

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(s.guid, "invalid score_then_tentatively_reserve request:", err)
			return
		}

		s.rep.ReleaseReservation(inst) //need to handle error

		responsePayload = successResponse
	})

	s.client.Subscribe(subjects.Run, func(msg *yagnats.Message) {
		var inst models.LRPStartAuction

		responsePayload := errorResponse
		defer func() {
			s.client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(s.guid, "invalid score_then_tentatively_reserve request:", err)
			return
		}

		s.rep.Run(inst) //need to handle error

		responsePayload = successResponse
	})

}

func (s *RepNatsServer) stop(subjects nats.Subjects) {
	for _, topic := range subjects.Slice() {
		s.client.UnsubscribeAll(topic)
	}
}
