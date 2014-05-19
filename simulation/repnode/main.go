package main

import (
	"flag"
	"strings"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats/repnatsserver"
	"github.com/cloudfoundry-incubator/auction/communication/rabbit/reprabbitserver"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
)

var memoryMB = flag.Float64("memoryMB", 100.0, "total available memory in MB")
var diskMB = flag.Float64("diskMB", 100.0, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var guid = flag.String("guid", "", "guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var rabbitAddr = flag.String("rabbitAddr", "", "rabbit server address")

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddrs == "" && *rabbitAddr == "" {
		panic("need nats or rabbit addr")
	}

	repDelegate := simulationrepdelegate.New(auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})
	rep := auctionrep.New(*guid, repDelegate)

	if *natsAddrs != "" {
		go repnatsserver.Start(strings.Split(*natsAddrs, ","), rep)
	}

	if *rabbitAddr != "" {
		go reprabbitserver.Start(*rabbitAddr, rep)
	}

	select {}
}
