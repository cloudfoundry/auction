package nats_muxer_test

import (
	"os"

	"github.com/cloudfoundry/gunk/diegonats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"

	"testing"
)

var natsProcess ifrit.Process
var natsClient diegonats.NATSClient

func TestNatsmuxer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Natsmuxer Suite")
}

var _ = BeforeEach(func() {
	natsProcess, natsClient = diegonats.StartGnatsd(GinkgoParallelNode() + 4001)
})

var _ = AfterEach(func() {
	natsClient.Close()
	natsProcess.Signal(os.Interrupt)
	Eventually(natsProcess.Wait(), 5).Should(Receive())
})
