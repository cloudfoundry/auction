package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_client"
	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry/gunk/diegonats"
)

var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var timeout = flag.Duration("timeout", time.Second, "timeout for nats responses")
var maxConcurrent = flag.Int("maxConcurrent", 1000, "number of concurrent auctions to hold")
var httpAddr = flag.String("httpAddr", "0.0.0.0:48710", "http address to listen on")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddrs == "" {
		panic("need nats addr")
	}

	if *httpAddr == "" {
		panic("need http addr")
	}

	var repClient auctiontypes.RepPoolClient

	natsMembers := []string{}
	for _, addr := range strings.Split(*natsAddrs, ",") {
		uri := url.URL{
			Scheme: "nats",
			Host:   addr,
		}
		natsMembers = append(natsMembers, uri.String())
	}
	client := diegonats.NewClient()
	_, err := client.Connect(natsMembers)
	if err != nil {
		log.Fatalln("no nats:", err)
	}

	repClient, err = auction_nats_client.New(client, *timeout, cf_lager.New("simulation"))
	if err != nil {
		log.Fatalln("no rep client:", err)
	}

	semaphore := make(chan bool, *maxConcurrent)

	http.HandleFunc("/start-auction", func(w http.ResponseWriter, r *http.Request) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest auctiontypes.StartAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStartAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	http.HandleFunc("/stop-auction", func(w http.ResponseWriter, r *http.Request) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest auctiontypes.StopAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStopAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe(*httpAddr, nil))
}
