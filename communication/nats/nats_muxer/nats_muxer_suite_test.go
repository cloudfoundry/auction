package nats_muxer_test

import (
	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var natsRunner *natsrunner.NATSRunner
var natsClient yagnats.NATSClient

func TestNatsmuxer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Natsmuxer Suite")
}

var _ = BeforeSuite(func() {
	natsRunner = natsrunner.NewNATSRunner(GinkgoParallelNode() + 4001)
})

var _ = BeforeEach(func() {
	natsRunner.Start()
	natsClient = natsRunner.MessageBus
})

var _ = AfterEach(func() {
	natsRunner.Stop()
})

var _ = AfterSuite(func() {
	natsRunner.KillWithFire()
})
