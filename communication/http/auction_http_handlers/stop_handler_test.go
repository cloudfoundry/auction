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

var _ = Describe("StopHandler", func() {
	Context("with valid JSON", func() {
		var stopLRPInstance models.StopLRPInstance

		BeforeEach(func() {
			stopLRPInstance = models.StopLRPInstance{
				ProcessGuid:  "process-guid",
				InstanceGuid: "instance-guid",
				Index:        1,
			}
		})

		It("should notify the auction rep", func() {
			Request(routes.Stop, nil, JSONReaderFor(stopLRPInstance))
			Ω(auctionRep.StopCallCount()).Should(Equal(1))
			Ω(auctionRep.StopArgsForCall(0)).Should(Equal(stopLRPInstance))
		})

		Context("and a succesful release", func() {
			BeforeEach(func() {
				auctionRep.StopReturns(nil)
			})

			It("should return success", func() {
				status, body := Request(routes.Stop, nil, JSONReaderFor(stopLRPInstance))
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(BeEmpty())
			})
		})

		Context("and an unsuccesful release", func() {
			BeforeEach(func() {
				auctionRep.StopReturns(errors.New("oops"))
			})

			It("should return a non-happy status code and the error", func() {
				status, body := Request(routes.Stop, nil, JSONReaderFor(stopLRPInstance))
				Ω(status).Should(Equal(http.StatusForbidden))
				Ω(body).Should(ContainSubstring("oops"))
			})
		})
	})

	Context("when invalid JSON", func() {
		It("should return an error without calling the rep", func() {
			status, body := Request(routes.Stop, nil, bytes.NewBufferString("∆"))
			Ω(status).Should(Equal(http.StatusBadRequest))
			Ω(body).Should(ContainSubstring("invalid json: invalid character"))

			Ω(auctionRep.StopCallCount()).Should(BeZero())
		})
	})
})
