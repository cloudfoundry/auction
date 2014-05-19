package reprabbitclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/rabbit/rabbitclient"
	"github.com/cloudfoundry-incubator/auction/util"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepRabbitClient struct {
	client  rabbitclient.RabbitClientInterface
	timeout time.Duration
}

func New(rabbitUrl string, timeout time.Duration) *RepRabbitClient {
	guid := util.RandomGuid()
	client := rabbitclient.NewClient(guid, rabbitUrl)
	err := client.ConnectAndEstablish()
	if err != nil {
		panic(err)
	}

	return &RepRabbitClient{
		client:  client,
		timeout: timeout,
	}
}

func (rep *RepRabbitClient) request(guid string, subject string, req interface{}, resp interface{}) (err error) {
	payload := []byte{}
	if req != nil {
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	response, err := rep.client.Request(guid, subject, payload, rep.timeout)

	if err != nil {
		return err
	}

	if string(response) == "error" {
		return RequestFailedError
	}

	if resp != nil {
		return json.Unmarshal(response, resp)
	}

	return nil
}

func (rep *RepRabbitClient) TotalResources(guid string) auctiontypes.Resources {
	var totalResources auctiontypes.Resources
	err := rep.request(guid, "total_resources", []byte{}, &totalResources)
	if err != nil {
		panic(err)
	}
	return totalResources
}

func (rep *RepRabbitClient) Instances(guid string) []auctiontypes.Instance {
	var instances []auctiontypes.Instance
	err := rep.request(guid, "instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepRabbitClient) Reset(guid string) {
	err := rep.request(guid, "reset", nil, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepRabbitClient) SetInstances(guid string, instances []auctiontypes.Instance) {
	err := rep.request(guid, "set_instances", instances, nil)
	if err != nil {
		panic(err)
	}
}

func (rep *RepRabbitClient) batch(subject string, guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	c := make(chan auctiontypes.ScoreResult)
	for _, guid := range guids {
		go func(guid string) {
			var response auctiontypes.ScoreResult
			err := rep.request(guid, subject, instance, &response)
			if err != nil {
				c <- auctiontypes.ScoreResult{
					Error: err.Error(),
				}
			}
			c <- response
		}(guid)
	}

	scores := auctiontypes.ScoreResults{}
	for _ = range guids {
		scores = append(scores, <-c)
	}

	return scores
}

func (rep *RepRabbitClient) Score(guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	return rep.batch("score", guids, instance)
}

func (rep *RepRabbitClient) ScoreThenTentativelyReserve(guids []string, instance auctiontypes.Instance) auctiontypes.ScoreResults {
	return rep.batch("score_then_tentatively_reserve", guids, instance)
}

func (rep *RepRabbitClient) ReleaseReservation(guids []string, instance auctiontypes.Instance) {
	allReceived := new(sync.WaitGroup)
	allReceived.Add(len(guids))
	for _, guid := range guids {
		go func(guid string) {
			rep.request(guid, "release-reservation", instance, nil)
			allReceived.Done()
		}(guid)
	}

	allReceived.Wait()
}

func (rep *RepRabbitClient) Claim(guid string, instance auctiontypes.Instance) {
	err := rep.request(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
