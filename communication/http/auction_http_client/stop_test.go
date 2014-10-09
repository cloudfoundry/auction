package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stop", func() {
	var stopLRPInstance models.StopLRPInstance

	BeforeEach(func() {
		stopLRPInstance = models.StopLRPInstance{
			ProcessGuid:  "process-guid",
			InstanceGuid: "instance-guid",
			Index:        1,
		}

		auctionRepA.StopReturns(nil)
		auctionRepB.StopReturns(errors.New("oops"))
	})

	It("should tell the rep to stop", func() {
		client.Stop(RepAddressFor("A"), stopLRPInstance)

		Ω(auctionRepA.StopCallCount()).Should(Equal(1))
		Ω(auctionRepA.StopArgsForCall(0)).Should(Equal(stopLRPInstance))

		client.Stop(RepAddressFor("B"), stopLRPInstance)

		Ω(auctionRepB.StopCallCount()).Should(Equal(1))
		Ω(auctionRepB.StopArgsForCall(0)).Should(Equal(stopLRPInstance))

		//these should not panic/explode
		client.Stop(RepAddressFor("RepThat500s"), stopLRPInstance)
		client.Stop(RepAddressFor("RepThatErrors"), stopLRPInstance)

	})

	PIt("what about errors?", func() {

	})
})
