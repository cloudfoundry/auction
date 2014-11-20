package simulation_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/cloudfoundry/gunk/timeprovider"

	"github.com/cloudfoundry/gunk/workpool"
	"github.com/tedsuo/ifrit"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrep"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
	"github.com/cloudfoundry-incubator/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/lager"

	"testing"
	"time"
)

var communicationMode string

const InProcess = "inprocess"
const HTTP = "http"

const numCells = 100

var cells map[string]auctiontypes.SimulationAuctionRep

var repResources = auctiontypes.Resources{
	MemoryMB:   100.0,
	DiskMB:     100.0,
	Containers: 100,
}

var timeout time.Duration
var workers int

var svgReport *visualization.SVGReport
var reports []*visualization.Report
var reportName string
var disableSVGReport bool

var sessionsToTerminate []*gexec.Session
var auctionRunnerProcess ifrit.Process
var auctionRunnerDelegate *AuctionRunnerDelegate
var auctionWorkPool *workpool.WorkPool
var auctionRunner auctionrunner.AuctionRunner

func init() {
	flag.StringVar(&communicationMode, "communicationMode", "inprocess", "one of inprocess, nats, or http")
	flag.DurationVar(&timeout, "timeout", time.Second, "timeout when waiting for responses from remote calls")
	flag.IntVar(&workers, "workers", 500, "number of concurrent communication worker pools")

	flag.BoolVar(&disableSVGReport, "disableSVGReport", false, "disable displaying SVG reports of the simulation runs")
	flag.StringVar(&reportName, "reportName", "report", "report name")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("Running in %s communicationMode\n", communicationMode)

	startReport()

	logger := lager.NewLogger("auction-sim")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

	sessionsToTerminate = []*gexec.Session{}
	switch communicationMode {
	case InProcess:
		cells = buildInProcessReps()
	case HTTP:
		cells = launchExternalHTTPReps()
	default:
		panic(fmt.Sprintf("unknown communication mode: %s", communicationMode))
	}
})

var _ = BeforeEach(func() {
	workPool := workpool.NewWorkPool(50)
	wg := &sync.WaitGroup{}
	wg.Add(len(cells))
	for _, cell := range cells {
		cell := cell
		workPool.Submit(func() {
			cell.Reset()
			wg.Done()
		})
	}

	wg.Wait()
	workPool.Stop()

	util.ResetGuids()

	auctionRunnerDelegate = NewAuctionRunnerDelegate(cells)
	auctionWorkPool = workpool.NewWorkPool(workers)
	auctionRunner = auctionrunner.New(auctionRunnerDelegate, timeprovider.NewTimeProvider(), 5, auctionWorkPool)
	auctionRunnerProcess = ifrit.Invoke(r)
})

var _ = AfterEach(func() {
	auctionRunnerProcess.Signal(os.Interrupt)
	Eventually(auctionRunnerProcess.Wait(), 20).Should(Receive())
	auctionWorkPool.Stop()
})

var _ = AfterSuite(func() {
	if !disableSVGReport {
		finishReport()
	}

	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}
})

func cellGuid(index int) string {
	return fmt.Sprintf("REP-%d", index+1)
}

func buildInProcessReps() map[string]auctiontypes.SimulationAuctionRep {
	cells := map[string]auctiontypes.SimulationAuctionRep{}

	for i := 0; i < numCells; i++ {
		cells[cellGuid(i)] = simulationrep.New("lucid64", repResources)
	}

	return cells
}

func launchExternalHTTPReps() map[string]auctiontypes.SimulationAuctionRep {
	panic("not yet!")
	return nil
	// repNodeBinary, err := gexec.Build("github.com/cloudfoundry-incubator/auction/simulation/repnode")
	// Ω(err).ShouldNot(HaveOccurred())

	// repAddresses := []auctiontypes.RepAddress{}

	// for i := 0; i < numCells; i++ {
	// 	repGuid := util.NewGuid("REP")
	// 	httpAddr := fmt.Sprintf("127.0.0.1:%d", 30000+i)

	// 	serverCmd := exec.Command(
	// 		repNodeBinary,
	// 		"-repGuid", repGuid,
	// 		"-httpAddr", httpAddr,
	// 		"-memoryMB", fmt.Sprintf("%d", repResources.MemoryMB),
	// 		"-diskMB", fmt.Sprintf("%d", repResources.DiskMB),
	// 		"-containers", fmt.Sprintf("%d", repResources.Containers),
	// 	)

	// 	sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
	// 	Ω(err).ShouldNot(HaveOccurred())
	// 	sessionsToTerminate = append(sessionsToTerminate, sess)
	// 	Eventually(sess).Should(gbytes.Say("listening"))

	// 	repAddresses = append(repAddresses, auctiontypes.RepAddress{
	// 		RepGuid: repGuid,
	// 		Address: "http://" + httpAddr,
	// 	})
	// }

	// return repAddresses
}

func startReport() {
	svgReport = visualization.StartSVGReport("./"+reportName+".svg", 4, 4, numCells)
	svgReport.DrawHeader(communicationMode)
}

func finishReport() {
	svgReport.Done()
	_, err := exec.LookPath("rsvg-convert")
	if err == nil {
		exec.Command("rsvg-convert", "-h", "2000", "--background-color=#fff", "./"+reportName+".svg", "-o", "./"+reportName+".png").Run()
		exec.Command("open", "./"+reportName+".png").Run()
	}

	data, err := json.Marshal(reports)
	Ω(err).ShouldNot(HaveOccurred())
	ioutil.WriteFile("./"+reportName+".json", data, 0777)
}
