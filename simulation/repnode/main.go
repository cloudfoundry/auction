package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	auction_nats_server "github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_server"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry/gunk/diegonats"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"
)

var memoryMB = flag.Int("memoryMB", 100, "total available memory in MB")
var diskMB = flag.Int("diskMB", 100, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var repGuid = flag.String("repGuid", "", "rep-guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")

func main() {
	flag.Parse()

	if *repGuid == "" {
		panic("need rep-guid")
	}

	if *natsAddrs == "" {
		panic("need nats addr")
	}

	repDelegate := simulationrepdelegate.New(auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})
	rep := auctionrep.New(*repGuid, repDelegate)

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

		log.Println("starting rep nats server")
		natsRunner := auction_nats_server.New(client, rep, cf_lager.New("repnode").Session(*repGuid))
		monitor := ifrit.Envoke(sigmon.New(natsRunner))
		fmt.Println("rep node listening")
		err = <-monitor.Wait()
		if err != nil {
			println("NATS SERVER EXITED WITH ERROR: ", err.Error())
		}
	}

	select {}
}
