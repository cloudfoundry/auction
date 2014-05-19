package auctiondistributor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

type httpRemoteAuctions struct {
	hosts []string
}

func newHttpRemoteAuctions(hosts []string) *httpRemoteAuctions {
	return &httpRemoteAuctions{hosts}
}

func (h *httpRemoteAuctions) RemoteAuction(auctionRequest types.AuctionRequest) types.AuctionResult {
	host := h.hosts[util.R.Intn(len(h.hosts))]

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/auction", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("FAILED! TO AUCTION", err)
		return types.AuctionResult{
			Instance: auctionRequest.Instance,
		}
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var result types.AuctionResult
	json.Unmarshal(data, &result)

	return result
}
