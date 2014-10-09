package auction_http_client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type AuctionHTTPClient struct {
	client *http.Client
	logger lager.Logger
}

type Response struct {
	Body []byte
}

func New(client *http.Client, logger lager.Logger) *AuctionHTTPClient {
	return &AuctionHTTPClient{
		client: client,
		logger: logger,
	}
}

func (c *AuctionHTTPClient) BidForStartAuction(repAddresses []auctiontypes.RepAddress, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	logger := c.logger.Session("bid-for-start-auction", lager.Data{
		"process-guid":  startAuctionInfo.ProcessGuid,
		"instance-guid": startAuctionInfo.InstanceGuid,
		"disk-mb":       startAuctionInfo.DiskMB,
		"memory-mb":     startAuctionInfo.MemoryMB,
		"index":         startAuctionInfo.Index,
	})
	logger = logger.WithData(lager.Data{
		"num-requests": len(repAddresses),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(startAuctionInfo)
	responses := c.batch(repAddresses, "GET", "/bids/start_auction", body)

	startAuctionBids := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		startAuctionBid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response.Body, &startAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		startAuctionBids = append(startAuctionBids, startAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(startAuctionBids),
	})
	logger.Info("done")

	return startAuctionBids
}

func (c *AuctionHTTPClient) BidForStopAuction(repAddresses []auctiontypes.RepAddress, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	logger := c.logger.Session("bid-for-stop-auction", lager.Data{
		"process-guid": stopAuctionInfo.ProcessGuid,
		"index":        stopAuctionInfo.Index,
		"num-requests": len(repAddresses),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(stopAuctionInfo)
	responses := c.batch(repAddresses, "GET", "/bids/stop_auction", body)

	stopAuctionBids := auctiontypes.StopAuctionBids{}
	for _, response := range responses {
		stopAuctionBid := auctiontypes.StopAuctionBid{}
		err := json.Unmarshal(response.Body, &stopAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		stopAuctionBids = append(stopAuctionBids, stopAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(stopAuctionBids),
	})
	logger.Info("done")

	return stopAuctionBids
}

func (c *AuctionHTTPClient) RebidThenTentativelyReserve(repAddresses []auctiontypes.RepAddress, startAuction models.LRPStartAuction) auctiontypes.StartAuctionBids {
	logger := c.logger.Session("rebid-then-tentatively-reserve", lager.Data{
		"process-guid":  startAuction.DesiredLRP.ProcessGuid,
		"instance-guid": startAuction.InstanceGuid,
		"disk-mb":       startAuction.DesiredLRP.DiskMB,
		"memory-mb":     startAuction.DesiredLRP.MemoryMB,
		"index":         startAuction.Index,
	})
	logger = logger.WithData(lager.Data{
		"num-requests": len(repAddresses),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(startAuction)
	responses := c.batch(repAddresses, "POST", "/reservations", body)

	startAuctionBids := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		startAuctionBid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response.Body, &startAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		startAuctionBids = append(startAuctionBids, startAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(startAuctionBids),
	})
	logger.Info("done")

	return startAuctionBids
}

func (c *AuctionHTTPClient) ReleaseReservation(repAddresses []auctiontypes.RepAddress, startAuction models.LRPStartAuction) {
	logger := c.logger.Session("release-reservation", lager.Data{
		"process-guid":  startAuction.DesiredLRP.ProcessGuid,
		"instance-guid": startAuction.InstanceGuid,
		"disk-mb":       startAuction.DesiredLRP.DiskMB,
		"memory-mb":     startAuction.DesiredLRP.MemoryMB,
		"index":         startAuction.Index,
	})
	logger.Info("requesting")
	body, _ := json.Marshal(startAuction)
	c.batch(repAddresses, "DELETE", "/reservations", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) Run(repAddress auctiontypes.RepAddress, lrpStartAuction models.LRPStartAuction) {
	logger := c.logger.Session("run", lager.Data{
		"process-guid":  lrpStartAuction.DesiredLRP.ProcessGuid,
		"instance-guid": lrpStartAuction.InstanceGuid,
		"index":         lrpStartAuction.Index,
	})
	logger.Info("requesting")
	body, _ := json.Marshal(lrpStartAuction)
	c.batch([]auctiontypes.RepAddress{repAddress}, "POST", "/run", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) Stop(repAddress auctiontypes.RepAddress, stopInstance models.StopLRPInstance) {
	logger := c.logger.Session("stop", lager.Data{
		"process-guid":  stopInstance.ProcessGuid,
		"instance-guid": stopInstance.InstanceGuid,
		"index":         stopInstance.Index,
	})
	logger.Info("requesting")
	body, _ := json.Marshal(stopInstance)
	c.batch([]auctiontypes.RepAddress{repAddress}, "POST", "/stop", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) TotalResources(repAddress auctiontypes.RepAddress) auctiontypes.Resources {
	responses := c.batch([]auctiontypes.RepAddress{repAddress}, "GET", "/sim/total_resources", nil)
	if len(responses) != 1 {
		return auctiontypes.Resources{}
	}
	resources := auctiontypes.Resources{}
	err := json.Unmarshal(responses[0].Body, &resources)
	if err != nil {
		return auctiontypes.Resources{}
	}
	return resources
}

func (c *AuctionHTTPClient) SimulatedInstances(repAddress auctiontypes.RepAddress) []auctiontypes.SimulatedInstance {
	responses := c.batch([]auctiontypes.RepAddress{repAddress}, "GET", "/sim/simulated_instances", nil)
	if len(responses) != 1 {
		return nil
	}
	instances := []auctiontypes.SimulatedInstance{}
	err := json.Unmarshal(responses[0].Body, &instances)
	if err != nil {
		return nil
	}
	return instances
}

func (c *AuctionHTTPClient) SetSimulatedInstances(repAddress auctiontypes.RepAddress, instances []auctiontypes.SimulatedInstance) {
	body, _ := json.Marshal(instances)
	c.batch([]auctiontypes.RepAddress{repAddress}, "POST", "/sim/simulated_instances", body)
}

func (c *AuctionHTTPClient) Reset(repAddress auctiontypes.RepAddress) {
	c.batch([]auctiontypes.RepAddress{repAddress}, "POST", "/sim/reset", nil)
}

/// batch http requests

func (c *AuctionHTTPClient) batch(repAddresses []auctiontypes.RepAddress, method string, path string, body []byte) []Response {
	requests := []*http.Request{}
	for _, repAddress := range repAddresses {
		reader := bytes.NewBuffer(body)
		url := repAddress.Address
		request, err := http.NewRequest(method, url+path, reader)
		if err != nil {
			continue
		}
		requests = append(requests, request)
	}

	return c.performRequests(requests)
}

func (c *AuctionHTTPClient) performRequests(requests []*http.Request) []Response {
	if len(requests) == 0 {
		return []Response{}
	}

	responsesChan := make(chan Response, len(requests))
	wg := &sync.WaitGroup{}
	wg.Add(len(requests))

	for _, request := range requests {
		go func(request *http.Request) {
			defer wg.Done()
			resp, err := c.client.Do(request)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			responsesChan <- Response{
				Body: responseBody,
			}
		}(request)
	}

	wg.Wait()
	close(responsesChan)
	responses := []Response{}
	for response := range responsesChan {
		responses = append(responses, response)
	}

	return responses
}
