package auction_http_handlers_test

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RunHandler", func() {
	Context("with valid JSON", func() {
		var lrpStartAuction models.LRPStartAuction

		BeforeEach(func() {
			lrpStartAuction = models.LRPStartAuction{
				InstanceGuid: "instance-guid",
				Index:        1,
			}
		})

		It("should notify the auction rep", func() {
			Request(routes.Run, nil, JSONReaderFor(lrpStartAuction))
			Ω(auctionRep.RunCallCount()).Should(Equal(1))
			Ω(auctionRep.RunArgsForCall(0)).Should(Equal(lrpStartAuction))
		})

		Context("and a succesful release", func() {
			BeforeEach(func() {
				auctionRep.RunReturns(nil)
			})

			It("should return success", func() {
				status, body := Request(routes.Run, nil, JSONReaderFor(lrpStartAuction))
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(BeEmpty())
			})
		})

		Context("and an unsuccesful release", func() {
			BeforeEach(func() {
				auctionRep.RunReturns(errors.New("oops"))
			})

			It("should return a non-happy status code and the error", func() {
				status, body := Request(routes.Run, nil, JSONReaderFor(lrpStartAuction))
				Ω(status).Should(Equal(http.StatusForbidden))
				Ω(body).Should(ContainSubstring("oops"))
			})
		})
	})

	Context("when invalid JSON", func() {
		It("should return an error without calling the rep", func() {
			status, body := Request(routes.Run, nil, bytes.NewBufferString("∆"))
			Ω(status).Should(Equal(http.StatusBadRequest))
			Ω(body).Should(ContainSubstring("invalid json: invalid character"))

			Ω(auctionRep.RunCallCount()).Should(BeZero())
		})
	})
})
