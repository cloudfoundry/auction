package nats_muxer_test

import (
	"github.com/cloudfoundry/gunk/diegonats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var natsRunner *diegonats.NATSRunner
var natsClient diegonats.NATSClient

func TestNatsmuxer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Natsmuxer Suite")
}

var _ = BeforeSuite(func() {
	natsRunner = diegonats.NewRunner(GinkgoParallelNode() + 4001)
})

var _ = BeforeEach(func() {
	natsRunner.Start()
	natsClient = natsRunner.Client
})

var _ = AfterEach(func() {
	natsRunner.Stop()
})

var _ = AfterSuite(func() {
	natsRunner.KillWithFire()
})
