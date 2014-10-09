package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReleaseReservation", func() {
	var lrpStartAuction models.LRPStartAuction

	BeforeEach(func() {
		lrpStartAuction = models.LRPStartAuction{
			InstanceGuid: "instance-guid",
			Index:        1,
		}

		auctionRepA.ReleaseReservationReturns(nil)
		auctionRepB.ReleaseReservationReturns(errors.New("oops"))
	})

	It("should tell all the reps to release the reservation", func() {
		client.ReleaseReservation(RepAddressesFor("A", "B", "RepThat500s", "RepThatErrors"), lrpStartAuction)

		Ω(auctionRepA.ReleaseReservationCallCount()).Should(Equal(1))
		Ω(auctionRepA.ReleaseReservationArgsForCall(0)).Should(Equal(lrpStartAuction))

		Ω(auctionRepB.ReleaseReservationCallCount()).Should(Equal(1))
		Ω(auctionRepB.ReleaseReservationArgsForCall(0)).Should(Equal(lrpStartAuction))
	})

	PIt("what about errors?", func() {

	})
})
