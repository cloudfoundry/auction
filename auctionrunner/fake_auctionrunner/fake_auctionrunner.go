package fake_auctionrunner

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type FakeAuctionRunner struct {
	lock                     *sync.Mutex
	auctionRequest           auctiontypes.StartAuctionRequest
	runLRPStartAuctionResult auctiontypes.StartAuctionResult
	runLRPStartAuctionError  error
	auctionDuration          time.Duration
}

func NewFakeAuctionRunner(auctionDuration time.Duration) *FakeAuctionRunner {
	return &FakeAuctionRunner{
		lock:            &sync.Mutex{},
		auctionDuration: auctionDuration,
	}
}

func (r *FakeAuctionRunner) RunLRPStartAuction(auctionRequest auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error) {
	r.lock.Lock()
	r.auctionRequest = auctionRequest
	r.lock.Unlock()

	time.Sleep(r.auctionDuration)

	r.lock.Lock()
	defer r.lock.Unlock()
	return r.runLRPStartAuctionResult, r.runLRPStartAuctionError
}

func (r *FakeAuctionRunner) SetStartAuctionResult(result auctiontypes.StartAuctionResult) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.runLRPStartAuctionResult = result
}

func (r *FakeAuctionRunner) SetStartAuctionError(err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.runLRPStartAuctionError = err
}

func (r *FakeAuctionRunner) GetStartAuctionRequest() auctiontypes.StartAuctionRequest {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.auctionRequest
}
