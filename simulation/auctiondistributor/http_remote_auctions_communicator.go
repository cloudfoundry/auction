package auctiondistributor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/util"
)

type httpRemoteAuctions struct {
	hosts []string
}

func newHttpRemoteAuctions(hosts []string) *httpRemoteAuctions {
	return &httpRemoteAuctions{hosts}
}

func (h *httpRemoteAuctions) RemoteAuction(auctionRequest auctiontypes.AuctionRequest) (auctiontypes.AuctionResult, error) {
	host := h.hosts[util.R.Intn(len(h.hosts))]

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/auction", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("FAILED! TO AUCTION", err)
		return auctiontypes.AuctionResult{
			LRPStartAuction: auctionRequest.LRPStartAuction,
		}, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return auctiontypes.AuctionResult{}, err
	}

	var result auctiontypes.AuctionResult
	json.Unmarshal(data, &result)

	return result, nil
}
