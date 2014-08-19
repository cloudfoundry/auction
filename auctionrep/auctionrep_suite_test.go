package auctionrep_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAuctionrep(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auctionrep Suite")
}
