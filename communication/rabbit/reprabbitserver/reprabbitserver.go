package reprabbitserver

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/rabbit/rabbitclient"
)

var errorResponse = []byte("error")
var successResponse = []byte("ok")

func Start(rabbitUrl string, rep *auctionrep.AuctionRep) {
	println("RABBIT", rabbitUrl)
	server := rabbitclient.NewServer(rep.Guid(), rabbitUrl)
	err := server.ConnectAndEstablish()
	if err != nil {
		panic(err)
	}

	server.Handle("total_resources", func(_ []byte) []byte {
		out, _ := json.Marshal(rep.TotalResources())
		return out
	})

	server.Handle("reset", func(_ []byte) []byte {
		rep.Reset()
		return successResponse
	})

	server.Handle("set_instances", func(req []byte) []byte {
		var instances []auctiontypes.Instance

		err := json.Unmarshal(req, &instances)
		if err != nil {
			return errorResponse
		}

		rep.SetInstances(instances)
		return successResponse
	})

	server.Handle("instances", func(_ []byte) []byte {
		out, _ := json.Marshal(rep.Instances())
		return out
	})

	server.Handle("score", func(req []byte) []byte {
		var inst auctiontypes.Instance

		err := json.Unmarshal(req, &inst)
		if err != nil {
			return errorResponse
		}

		response := auctiontypes.ScoreResult{
			Rep: rep.Guid(),
		}

		score, err := rep.Score(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Score = score
		}

		out, _ := json.Marshal(response)
		return out
	})

	server.Handle("score_then_tentatively_reserve", func(req []byte) []byte {
		var inst auctiontypes.Instance

		err := json.Unmarshal(req, &inst)
		if err != nil {
			return errorResponse
		}

		response := auctiontypes.ScoreResult{
			Rep: rep.Guid(),
		}

		score, err := rep.ScoreThenTentativelyReserve(inst)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Score = score
		}

		out, _ := json.Marshal(response)
		return out
	})

	server.Handle("release-reservation", func(req []byte) []byte {
		var instance auctiontypes.Instance

		err := json.Unmarshal(req, &instance)
		if err != nil {
			return errorResponse
		}

		rep.ReleaseReservation(instance)

		return successResponse
	})

	server.Handle("claim", func(req []byte) []byte {
		var instance auctiontypes.Instance

		err := json.Unmarshal(req, &instance)
		if err != nil {
			return errorResponse
		}

		rep.Claim(instance)

		return successResponse
	})

	fmt.Printf("[%s] listening for rabbit\n", rep.Guid())

	select {}
}
