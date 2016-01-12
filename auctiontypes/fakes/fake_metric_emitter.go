// This file was generated by counterfeiter
package fakes

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type FakeAuctionMetricEmitterDelegate struct {
	FetchStatesCompletedStub        func(time.Duration) error
	fetchStatesCompletedMutex       sync.RWMutex
	fetchStatesCompletedArgsForCall []struct {
		arg1 time.Duration
	}
	fetchStatesCompletedReturns struct {
		result1 error
	}
	FailedCellStateRequestStub        func()
	failedCellStateRequestMutex       sync.RWMutex
	failedCellStateRequestArgsForCall []struct{}
	AuctionCompletedStub        func(auctiontypes.AuctionResults)
	auctionCompletedMutex       sync.RWMutex
	auctionCompletedArgsForCall []struct {
		arg1 auctiontypes.AuctionResults
	}
}

func (fake *FakeAuctionMetricEmitterDelegate) FetchStatesCompleted(arg1 time.Duration) error {
	fake.fetchStatesCompletedMutex.Lock()
	fake.fetchStatesCompletedArgsForCall = append(fake.fetchStatesCompletedArgsForCall, struct {
		arg1 time.Duration
	}{arg1})
	fake.fetchStatesCompletedMutex.Unlock()
	if fake.FetchStatesCompletedStub != nil {
		return fake.FetchStatesCompletedStub(arg1)
	} else {
		return fake.fetchStatesCompletedReturns.result1
	}
}

func (fake *FakeAuctionMetricEmitterDelegate) FetchStatesCompletedCallCount() int {
	fake.fetchStatesCompletedMutex.RLock()
	defer fake.fetchStatesCompletedMutex.RUnlock()
	return len(fake.fetchStatesCompletedArgsForCall)
}

func (fake *FakeAuctionMetricEmitterDelegate) FetchStatesCompletedArgsForCall(i int) time.Duration {
	fake.fetchStatesCompletedMutex.RLock()
	defer fake.fetchStatesCompletedMutex.RUnlock()
	return fake.fetchStatesCompletedArgsForCall[i].arg1
}

func (fake *FakeAuctionMetricEmitterDelegate) FetchStatesCompletedReturns(result1 error) {
	fake.FetchStatesCompletedStub = nil
	fake.fetchStatesCompletedReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeAuctionMetricEmitterDelegate) FailedCellStateRequest() {
	fake.failedCellStateRequestMutex.Lock()
	fake.failedCellStateRequestArgsForCall = append(fake.failedCellStateRequestArgsForCall, struct{}{})
	fake.failedCellStateRequestMutex.Unlock()
	if fake.FailedCellStateRequestStub != nil {
		fake.FailedCellStateRequestStub()
	}
}

func (fake *FakeAuctionMetricEmitterDelegate) FailedCellStateRequestCallCount() int {
	fake.failedCellStateRequestMutex.RLock()
	defer fake.failedCellStateRequestMutex.RUnlock()
	return len(fake.failedCellStateRequestArgsForCall)
}

func (fake *FakeAuctionMetricEmitterDelegate) AuctionCompleted(arg1 auctiontypes.AuctionResults) {
	fake.auctionCompletedMutex.Lock()
	fake.auctionCompletedArgsForCall = append(fake.auctionCompletedArgsForCall, struct {
		arg1 auctiontypes.AuctionResults
	}{arg1})
	fake.auctionCompletedMutex.Unlock()
	if fake.AuctionCompletedStub != nil {
		fake.AuctionCompletedStub(arg1)
	}
}

func (fake *FakeAuctionMetricEmitterDelegate) AuctionCompletedCallCount() int {
	fake.auctionCompletedMutex.RLock()
	defer fake.auctionCompletedMutex.RUnlock()
	return len(fake.auctionCompletedArgsForCall)
}

func (fake *FakeAuctionMetricEmitterDelegate) AuctionCompletedArgsForCall(i int) auctiontypes.AuctionResults {
	fake.auctionCompletedMutex.RLock()
	defer fake.auctionCompletedMutex.RUnlock()
	return fake.auctionCompletedArgsForCall[i].arg1
}

var _ auctiontypes.AuctionMetricEmitterDelegate = new(FakeAuctionMetricEmitterDelegate)
