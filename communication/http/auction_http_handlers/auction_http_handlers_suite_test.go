package auction_http_handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/rata"

	"testing"
)

func TestAuctionHttpHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AuctionHttpHandlers Suite")
}

var server *httptest.Server
var requestGenerator *rata.RequestGenerator
var client *http.Client
var auctionRep *fakes.FakeSimulationAuctionRep
var repGuid string

var _ = BeforeEach(func() {
	logger := lagertest.NewTestLogger("auction_http_handlers")

	auctionRep = &fakes.FakeSimulationAuctionRep{}
	repGuid = "alpha-omicron-delta"
	auctionRep.GuidReturns(repGuid)

	handler, err := rata.NewRouter(routes.Routes, auction_http_handlers.New(auctionRep, logger))
	Ω(err).ShouldNot(HaveOccurred())
	server = httptest.NewServer(handler)

	requestGenerator = rata.NewRequestGenerator(server.URL, routes.Routes)

	client = &http.Client{}
})

var _ = AfterEach(func() {
	server.Close()
})

func JSONFor(obj interface{}) string {
	marshalled, err := json.Marshal(obj)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return string(marshalled)
}

func JSONReaderFor(obj interface{}) io.Reader {
	marshalled, err := json.Marshal(obj)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return bytes.NewBuffer(marshalled)
}

func Request(name string, params rata.Params, body io.Reader) (statusCode int, responseBody []byte) {
	request, err := requestGenerator.CreateRequest(name, params, body)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	response, err := client.Do(request)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	responseBody, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return response.StatusCode, responseBody
}
