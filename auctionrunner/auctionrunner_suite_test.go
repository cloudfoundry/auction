package auctionrunner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
)

var logger lager.Logger

func TestAuctionrunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auctionrunner Suite")
}

var _ = BeforeEach(func() {
	logger = lagertest.NewTestLogger("test")
})
