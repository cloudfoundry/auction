package auction_http_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

type AuctionHTTPClient struct {
	client           *http.Client
	repGuid          string
	address          string
	requestGenerator *rata.RequestGenerator
	logger           lager.Logger
}

type Response struct {
	Body []byte
}

func New(client *http.Client, repGuid string, address string, logger lager.Logger) *AuctionHTTPClient {
	return &AuctionHTTPClient{
		client:           client,
		repGuid:          repGuid,
		address:          address,
		requestGenerator: rata.NewRequestGenerator(address, routes.Routes),
		logger:           logger,
	}
}

func (c *AuctionHTTPClient) State() (auctiontypes.CellState, error) {
	logger := c.logger.Session("fetching-state", lager.Data{
		"rep": c.repGuid,
	})

	logger.Debug("requesting")

	req, err := c.requestGenerator.CreateRequest(routes.State, nil, nil)
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return auctiontypes.CellState{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return auctiontypes.CellState{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return auctiontypes.CellState{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var state auctiontypes.CellState
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		logger.Error("failed-to-decode-rep-state", err)
		return auctiontypes.CellState{}, err
	}

	logger.Debug("done")

	return state, nil
}

func (c *AuctionHTTPClient) Perform(work auctiontypes.Work) (auctiontypes.Work, error) {
	logger := c.logger.Session("sending-work", lager.Data{
		"rep":    c.repGuid,
		"starts": len(work.LRPs),
	})

	logger.Debug("requesting")

	body, err := json.Marshal(work)
	if err != nil {
		logger.Error("failed-to-marshal-work", err)
		return auctiontypes.Work{}, err
	}

	req, err := c.requestGenerator.CreateRequest(routes.Perform, nil, bytes.NewReader(body))
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return auctiontypes.Work{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return auctiontypes.Work{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return auctiontypes.Work{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var failedWork auctiontypes.Work
	err = json.NewDecoder(resp.Body).Decode(&failedWork)
	if err != nil {
		logger.Error("failed-to-decode-failed-work", err)
		return auctiontypes.Work{}, err
	}

	logger.Debug("done")

	return failedWork, nil
}

func (c *AuctionHTTPClient) Reset() error {
	logger := c.logger.Session("SIM-reseting", lager.Data{
		"rep": c.repGuid,
	})

	logger.Debug("requesting")

	req, err := c.requestGenerator.CreateRequest(routes.Sim_Reset, nil, nil)
	if err != nil {
		logger.Error("failed-to-create-request", err)
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.Error("failed-to-perform-request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("invalid-status-code", fmt.Errorf("%d", resp.StatusCode))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	logger.Debug("done")
	return nil
}
