package auction_http_handlers_test

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Perform", func() {
	Context("with valid JSON", func() {
		var requestedWork, failedWork auctiontypes.Work
		BeforeEach(func() {
			requestedWork = auctiontypes.Work{
				LRPStops: []models.ActualLRP{
					{
						ProcessGuid:  "pg-a",
						InstanceGuid: "ig-a",
						Index:        1,
						CellID:       "A",
					},
					{
						ProcessGuid:  "pg-b",
						InstanceGuid: "ig-b",
						Index:        2,
						CellID:       "B",
					},
				},
			}

			failedWork = auctiontypes.Work{
				LRPStops: []models.ActualLRP{
					{
						ProcessGuid:  "pg-a",
						InstanceGuid: "ig-a",
						Index:        1,
						CellID:       "A",
					},
				},
			}
		})

		Context("and no perform error", func() {
			BeforeEach(func() {
				auctionRep.PerformReturns(failedWork, nil)
			})

			It("succeeds, returning any failed work", func() {
				Ω(auctionRep.PerformCallCount()).Should(Equal(0))

				status, body := Request(routes.Perform, nil, JSONReaderFor(requestedWork))
				Ω(status).Should(Equal(http.StatusOK))
				Ω(body).Should(MatchJSON(JSONFor(failedWork)))

				Ω(auctionRep.PerformCallCount()).Should(Equal(1))
				Ω(auctionRep.PerformArgsForCall(0)).Should(Equal(requestedWork))
			})
		})

		Context("and a perform error", func() {
			BeforeEach(func() {
				auctionRep.PerformReturns(failedWork, errors.New("kaboom"))
			})

			It("fails, returning nothing", func() {
				Ω(auctionRep.PerformCallCount()).Should(Equal(0))

				status, body := Request(routes.Perform, nil, JSONReaderFor(requestedWork))
				Ω(status).Should(Equal(http.StatusInternalServerError))
				Ω(body).Should(BeEmpty())

				Ω(auctionRep.PerformCallCount()).Should(Equal(1))
				Ω(auctionRep.PerformArgsForCall(0)).Should(Equal(requestedWork))
			})
		})
	})

	Context("with invalid JSON", func() {
		It("fails", func() {
			Ω(auctionRep.PerformCallCount()).Should(Equal(0))

			status, body := Request(routes.Perform, nil, bytes.NewBufferString("∆"))
			Ω(status).Should(Equal(http.StatusBadRequest))
			Ω(body).Should(BeEmpty())

			Ω(auctionRep.PerformCallCount()).Should(Equal(0))
		})
	})
})
