package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/tedsuo/rata"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	auction_nats_server "github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_server"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry/gunk/diegonats"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var memoryMB = flag.Int("memoryMB", 100, "total available memory in MB")
var diskMB = flag.Int("diskMB", 100, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var repGuid = flag.String("repGuid", "", "rep-guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var httpAddr = flag.String("httpAddr", "", "http server addres")

func main() {
	flag.Parse()

	if *repGuid == "" {
		panic("need rep-guid")
	}

	if *natsAddrs == "" && *httpAddr == "" {
		panic("need nats or http addr")
	}

	repDelegate := simulationrepdelegate.New(auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})
	rep := auctionrep.New(*repGuid, repDelegate)

	members := grouper.Members{}

	if *natsAddrs != "" {
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

		natsRunner := auction_nats_server.New(client, rep, cf_lager.New("repnode-nats").Session(*repGuid))
		members = append(members, grouper.Member{
			"nats-server",
			natsRunner,
		})
	}

	if *httpAddr != "" {
		handlers := auction_http_handlers.New(rep, cf_lager.New("repnode-http").Session(*repGuid))
		router, err := rata.NewRouter(routes.Routes, handlers)
		if err != nil {
			log.Fatalln("failed to make router:", err)
		}
		httpServer := http_server.New(*httpAddr, router)
		members = append(members, grouper.Member{
			"http-server",
			httpServer,
		})
	}

	if len(members) == 0 {
		log.Fatalln("no configured servers")
	}
	var monitor ifrit.Process
	if len(members) > 1 {
		monitor = ifrit.Envoke(sigmon.New(grouper.NewParallel(os.Interrupt, members)))
	} else {
		monitor = ifrit.Envoke(sigmon.New(members[0]))
	}
	fmt.Println("rep node listening")
	err := <-monitor.Wait()
	if err != nil {
		println("EXITED WITH ERROR: ", err.Error())
	}
}
