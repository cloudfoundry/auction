package simulation_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_client"
	"github.com/cloudfoundry-incubator/auction/simulation/auctiondistributor"
	"github.com/cloudfoundry-incubator/auction/simulation/communication/inprocess"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry/gunk/natsrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager"

	"testing"
	"time"
)

var communicationMode string
var auctioneerMode string

const InProcess = "inprocess"
const NATS = "nats"
const RemoteAuctioneerMode = "remote"

const numAuctioneers = 10
const numReps = 100
const LatencyMin = 50 * time.Millisecond
const LatencyMax = 100 * time.Millisecond

var repResources = auctiontypes.Resources{
	MemoryMB:   100.0,
	DiskMB:     100.0,
	Containers: 100,
}

var maxConcurrentPerExecutor int

var timeout time.Duration
var runTimeout time.Duration
var auctionDistributor *auctiondistributor.AuctionDistributor

var svgReport *visualization.SVGReport
var reports []*visualization.Report

var sessionsToTerminate []*gexec.Session
var natsRunner *natsrunner.NATSRunner
var client auctiontypes.SimulationRepPoolClient
var repGuids []string

var disableSVGReport bool

func init() {
	flag.StringVar(&communicationMode, "communicationMode", "inprocess", "one of inprocess or nats")
	flag.StringVar(&auctioneerMode, "auctioneerMode", "inprocess", "one of inprocess or remote")
	flag.DurationVar(&timeout, "timeout", 500*time.Millisecond, "timeout when waiting for responses from remote calls")
	flag.DurationVar(&runTimeout, "runTimeout", 10*time.Second, "timeout when waiting for the run command to respond")

	flag.StringVar(&(auctionrunner.DefaultStartAuctionRules.Algorithm), "algorithm", auctionrunner.DefaultStartAuctionRules.Algorithm, "the auction algorithm to use")
	flag.IntVar(&(auctionrunner.DefaultStartAuctionRules.MaxRounds), "maxRounds", auctionrunner.DefaultStartAuctionRules.MaxRounds, "the maximum number of rounds per auction")
	flag.Float64Var(&(auctionrunner.DefaultStartAuctionRules.MaxBiddingPoolFraction), "maxBiddingPoolFraction", auctionrunner.DefaultStartAuctionRules.MaxBiddingPoolFraction, "the maximum number of participants in the pool")

	flag.IntVar(&maxConcurrentPerExecutor, "maxConcurrentPerExecutor", 20, "the maximum number of concurrent auctions to run, per executor")

	flag.BoolVar(&disableSVGReport, "disableSVGReport", false, "disable displaying SVG reports of the simulation runs")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("Running in %s communicationMode\n", communicationMode)
	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	startReport()

	sessionsToTerminate = []*gexec.Session{}
	hosts := []string{}
	switch communicationMode {
	case InProcess:
		client, repGuids = buildInProcessReps()
		if auctioneerMode == RemoteAuctioneerMode {
			panic("it doesn't make sense to use remote auctioneers when the reps are in-process")
		}
	case NATS:
		natsAddrs := startNATS()
		var err error

		natsLogger := lager.NewLogger("test")
		natsLogger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		client, err = auction_nats_client.New(natsRunner.MessageBus, timeout, runTimeout, natsLogger)
		Ω(err).ShouldNot(HaveOccurred())
		repGuids = launchExternalReps("-natsAddrs", natsAddrs)
		if auctioneerMode == RemoteAuctioneerMode {
			hosts = launchExternalAuctioneers("-natsAddrs", natsAddrs)
		}
	default:
		panic(fmt.Sprintf("unknown communication mode: %s", communicationMode))
	}

	if auctioneerMode == InProcess {
		auctionDistributor = auctiondistributor.NewInProcessAuctionDistributor(client)
	} else if auctioneerMode == RemoteAuctioneerMode {
		auctionDistributor = auctiondistributor.NewRemoteAuctionDistributor(hosts, client)
	}
})

var _ = BeforeEach(func() {
	for _, repGuid := range repGuids {
		client.Reset(repGuid)
	}

	util.ResetGuids()
})

var _ = AfterSuite(func() {
	if !disableSVGReport {
		finishReport()
	}
	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}

	if natsRunner != nil {
		natsRunner.Stop()
	}
})

func buildInProcessReps() (auctiontypes.SimulationRepPoolClient, []string) {
	inprocess.LatencyMin = LatencyMin
	inprocess.LatencyMax = LatencyMax

	repGuids := []string{}
	repMap := map[string]*auctionrep.AuctionRep{}

	for i := 0; i < numReps; i++ {
		repGuid := util.NewGuid("REP")
		repGuids = append(repGuids, repGuid)

		repDelegate := simulationrepdelegate.New(repResources)
		repMap[repGuid] = auctionrep.New(repGuid, repDelegate)
	}

	client := inprocess.New(repMap)
	return client, repGuids
}

func startNATS() string {
	natsPort := 5222 + GinkgoParallelNode()
	natsAddrs := []string{fmt.Sprintf("127.0.0.1:%d", natsPort)}
	natsRunner = natsrunner.NewNATSRunner(natsPort)
	natsRunner.Start()
	return strings.Join(natsAddrs, ",")
}

func launchExternalReps(communicationFlag string, communicationValue string) []string {
	repNodeBinary, err := gexec.Build("github.com/cloudfoundry-incubator/auction/simulation/repnode")
	Ω(err).ShouldNot(HaveOccurred())

	repGuids := []string{}

	for i := 0; i < numReps; i++ {
		repGuid := util.NewGuid("REP")

		serverCmd := exec.Command(
			repNodeBinary,
			"-repGuid", repGuid,
			communicationFlag, communicationValue,
			"-memoryMB", fmt.Sprintf("%d", repResources.MemoryMB),
			"-diskMB", fmt.Sprintf("%d", repResources.DiskMB),
			"-containers", fmt.Sprintf("%d", repResources.Containers),
		)

		sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(sess).Should(gbytes.Say("listening"))
		sessionsToTerminate = append(sessionsToTerminate, sess)

		repGuids = append(repGuids, repGuid)
	}

	return repGuids
}

func launchExternalAuctioneers(communicationFlag string, communicationValue string) []string {
	auctioneerNodeBinary, err := gexec.Build("github.com/cloudfoundry-incubator/auction/simulation/auctioneernode")
	Ω(err).ShouldNot(HaveOccurred())

	auctioneerHosts := []string{}
	for i := 0; i < numAuctioneers; i++ {
		port := 48710 + i
		auctioneerCmd := exec.Command(
			auctioneerNodeBinary,
			communicationFlag, communicationValue,
			"-timeout", fmt.Sprintf("%s", timeout),
			"-httpAddr", fmt.Sprintf("127.0.0.1:%d", port),
		)
		auctioneerHosts = append(auctioneerHosts, fmt.Sprintf("127.0.0.1:%d", port))

		sess, err := gexec.Start(auctioneerCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		Eventually(sess).Should(gbytes.Say("auctioneering"))
		sessionsToTerminate = append(sessionsToTerminate, sess)
	}

	return auctioneerHosts
}

func startReport() {
	svgReport = visualization.StartSVGReport("./report.svg", 4, 4)
	svgReport.DrawHeader(communicationMode, auctionrunner.DefaultStartAuctionRules, maxConcurrentPerExecutor)
}

func finishReport() {
	svgReport.Done()
	exec.Command("rsvg-convert", "-h", "2000", "--background-color=#fff", "./report.svg", "-o", "./report.png").Run()
	exec.Command("open", "./report.png").Run()

	data, err := json.Marshal(reports)
	Ω(err).ShouldNot(HaveOccurred())
	ioutil.WriteFile("./report.json", data, 0777)
}
