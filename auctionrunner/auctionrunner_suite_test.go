package auctionrunner_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAuctionrunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auctionrunner Suite")
}
