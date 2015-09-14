package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/auction/simulation/simulationrep"
	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	executorfakes "github.com/cloudfoundry-incubator/executor/fakes"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry-incubator/rep/evacuation/evacuation_context/fake_evacuation_context"
	rephandlers "github.com/cloudfoundry-incubator/rep/handlers"
	"github.com/cloudfoundry-incubator/rep/lrp_stopper/fake_lrp_stopper"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
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

	simulationRep := simulationrep.New(*stack, *zone, rep.Resources{
		MemoryMB:   int32(*memoryMB),
		DiskMB:     int32(*diskMB),
		Containers: *containers,
	})

	logger, _ := cf_lager.New("repnode-http")

	fakeLRPStopper := new(fake_lrp_stopper.FakeLRPStopper)
	fakeExecutorClient := new(executorfakes.FakeClient)
	fakeEvacuatable := new(fake_evacuation_context.FakeEvacuatable)

	handlers := rephandlers.New(simulationRep, fakeLRPStopper, fakeExecutorClient, fakeEvacuatable, logger.Session(*repGuid))
	router, err := rata.NewRouter(rep.Routes, handlers)
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
