package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/communication/nats/repnatsclient"
	"github.com/onsi/auction/communication/rabbit/reprabbitclient"
	"github.com/onsi/auction/types"
)

var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var rabbitAddr = flag.String("rabbitAddr", "", "rabbit server addresses")
var timeout = flag.Duration("timeout", 500*time.Millisecond, "timeout for entire auction")
var maxConcurrent = flag.Int("maxConcurrent", 1000, "number of concurrent auctions to hold")
var httpAddr = flag.String("httpAddr", "0.0.0.0:48710", "http address to listen on")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddrs == "" && *rabbitAddr == "" {
		panic("need nats or rabbit addr")
	}

	if *natsAddrs != "" && *rabbitAddr != "" {
		panic("can't have both nats and rabbit addrs, choose one")
	}

	if *httpAddr == "" {
		panic("need http addr")
	}

	var repClient types.RepPoolClient

	if *natsAddrs != "" {
		client := yagnats.NewClient()

		clusterInfo := &yagnats.ConnectionCluster{}

		for _, addr := range strings.Split(*natsAddrs, ",") {
			clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
				Addr: addr,
			})
		}

		err := client.Connect(clusterInfo)

		if err != nil {
			log.Fatalln("no nats:", err)
		}

		repClient = repnatsclient.New(client, *timeout)
	}

	if *rabbitAddr != "" {
		repClient = reprabbitclient.New(*rabbitAddr, *timeout)
	}

	semaphore := make(chan bool, *maxConcurrent)

	http.HandleFunc("/auction", func(w http.ResponseWriter, r *http.Request) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest types.AuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult := auctioneer.Auction(repClient, auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe(*httpAddr, nil))
}
