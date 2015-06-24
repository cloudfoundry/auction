package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/auction/simulation/simulationrep"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/tedsuo/rata"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var memoryMB = flag.Int("memoryMB", 100, "total available memory in MB")
var diskMB = flag.Int("diskMB", 100, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var repGuid = flag.String("repGuid", "", "rep-guid")
var httpAddr = flag.String("httpAddr", "", "http server addres")
var stack = flag.String("stack", "", "stack")
var zone = flag.String("zone", "Z0", "availability zone")

func main() {
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	if *repGuid == "" {
		panic("need rep-guid")
	}

	if *httpAddr == "" {
		panic("need http addr")
	}

	simulationRep := simulationrep.New(*stack, *zone, auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})

	logger, _ := cf_lager.New("repnode-http")
	handlers := auction_http_handlers.New(simulationRep, logger.Session(*repGuid))
	router, err := rata.NewRouter(routes.Routes, handlers)
	if err != nil {
		log.Fatalln("failed to make router:", err)
	}
	httpServer := http_server.New(*httpAddr, router)

	monitor := ifrit.Invoke(sigmon.New(httpServer))
	fmt.Println("rep node listening")
	err = <-monitor.Wait()
	if err != nil {
		println("EXITED WITH ERROR: ", err.Error())
	}
}
