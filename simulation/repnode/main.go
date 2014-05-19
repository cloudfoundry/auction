package main

import (
	"flag"
	"strings"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/communication/nats/repnatsserver"
	"github.com/onsi/auction/communication/rabbit/reprabbitserver"
	"github.com/onsi/auction/simulation/simulationrepdelegate"
	"github.com/onsi/auction/types"
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

	repDelegate := simulationrepdelegate.New(types.Resources{
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
