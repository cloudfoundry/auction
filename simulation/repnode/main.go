package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	repnatsserver "github.com/cloudfoundry-incubator/auction/communication/nats/repnatsserver"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
	"github.com/cloudfoundry/yagnats"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"
)

var memoryMB = flag.Int("memoryMB", 100, "total available memory in MB")
var diskMB = flag.Int("diskMB", 100, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var guid = flag.String("guid", "", "guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddrs == "" {
		panic("need nats addr")
	}

	repDelegate := simulationrepdelegate.New(auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})
	rep := auctionrep.New(*guid, repDelegate)

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

		log.Println("starting rep nats server")
		natsRunner := repnatsserver.New(client, rep)
		server := ifrit.Envoke(natsRunner)
		monitor := ifrit.Envoke(sigmon.New(server))
		err = <-monitor.Wait()
		if err != nil {
			println("NATS SERVER EXITED WITH ERROR: ", err.Error())
		}
	}

	select {}
}
