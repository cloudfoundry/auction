package repnatsserver

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/yagnats"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

func Start(client yagnats.NATSClient, rep *auctionrep.AuctionRep) {
	guid := rep.Guid()

	client.Subscribe(guid+".total_resources", func(msg *yagnats.Message) {
		jresources, _ := json.Marshal(rep.TotalResources())
		client.Publish(msg.ReplyTo, jresources)
	})

	client.Subscribe(guid+".reset", func(msg *yagnats.Message) {
		rep.Reset()
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".set_lrp_auction_infos", func(msg *yagnats.Message) {
		var instances []auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &instances)
		if err != nil {
			client.Publish(msg.ReplyTo, errorResponse)
		}

		rep.SetLRPAuctionInfos(instances)
		client.Publish(msg.ReplyTo, successResponse)
	})

	client.Subscribe(guid+".lrp_auction_infos", func(msg *yagnats.Message) {
		jinstances, _ := json.Marshal(rep.LRPAuctionInfos())
		client.Publish(msg.ReplyTo, jinstances)
	})

	client.Subscribe(guid+".score", func(msg *yagnats.Message) {
		var inst auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
		}

		response := auctiontypes.ScoreResult{
			Rep: guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			client.Publish(msg.ReplyTo, payload)
		}()

		score, err := rep.Score(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".score_then_tentatively_reserve", func(msg *yagnats.Message) {
		var inst auctiontypes.LRPAuctionInfo

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			panic(err)
		}

		response := auctiontypes.ScoreResult{
			Rep: guid,
		}

		defer func() {
			payload, _ := json.Marshal(response)
			client.Publish(msg.ReplyTo, payload)
		}()

		score, err := rep.ScoreThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
			return
		}

		response.Score = score
	})

	client.Subscribe(guid+".release-reservation", func(msg *yagnats.Message) {
		var inst auctiontypes.LRPAuctionInfo

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid score_then_tentatively_reserve request:", err)
			return
		}

		rep.ReleaseReservation(inst) //need to handle error

		responsePayload = successResponse
	})

	client.Subscribe(guid+".run", func(msg *yagnats.Message) {
		var inst models.LRPStartAuction

		responsePayload := errorResponse
		defer func() {
			client.Publish(msg.ReplyTo, responsePayload)
		}()

		err := json.Unmarshal(msg.Payload, &inst)
		if err != nil {
			log.Println(guid, "invalid score_then_tentatively_reserve request:", err)
			return
		}

		rep.Run(inst) //need to handle error

		responsePayload = successResponse
	})

	fmt.Printf("[%s] listening for nats\n", guid)
}
