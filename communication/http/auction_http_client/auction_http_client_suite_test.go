package auction_http_client_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"

	"testing"
)

func TestAuctionHttpClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AuctionHttpClient Suite")
}

var auctionRep *fakes.FakeSimulationCellRep
var server *httptest.Server
var serverThatErrors *ghttp.Server
var client, clientForServerThatErrors auctiontypes.SimulationCellRep

var _ = BeforeEach(func() {
	logger := lagertest.NewTestLogger("test")

	auctionRep = &fakes.FakeSimulationCellRep{}

	handler, err := rata.NewRouter(routes.Routes, auction_http_handlers.New(auctionRep, logger))
	Expect(err).NotTo(HaveOccurred())
	server = httptest.NewServer(handler)

	client = auction_http_client.New(&http.Client{}, "rep-guid", server.URL, logger)

	serverThatErrors = ghttp.NewServer()
	erroringHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		serverThatErrors.CloseClientConnections()
	})
	//5 erroringHandlers should be more than enough: none of the individual tests should make more than 5 requests to this server
	serverThatErrors.AppendHandlers(erroringHandler, erroringHandler, erroringHandler, erroringHandler, erroringHandler)

	clientForServerThatErrors = auction_http_client.New(&http.Client{}, "rep-guid", serverThatErrors.URL(), logger)
})

var _ = AfterEach(func() {
	server.Close()
	serverThatErrors.Close()
})
