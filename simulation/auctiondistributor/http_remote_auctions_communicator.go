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

func (h *httpRemoteAuctions) RemoteAuction(auctionRequest auctiontypes.AuctionRequest) auctiontypes.AuctionResult {
	host := h.hosts[util.R.Intn(len(h.hosts))]

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/auction", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("FAILED! TO AUCTION", err)
		return auctiontypes.AuctionResult{
			Instance: auctionRequest.Instance,
		}
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var result auctiontypes.AuctionResult
	json.Unmarshal(data, &result)

	return result
}
