// This file was generated by counterfeiter
package fakes

import (
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type FakeAuctionRunner struct {
	RunStub        func(signals <-chan os.Signal, ready chan<- struct{}) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}
	runReturns struct {
		result1 error
	}
	AddLRPStartAuctionStub        func(models.LRPStartAuction)
	addLRPStartAuctionMutex       sync.RWMutex
	addLRPStartAuctionArgsForCall []struct {
		arg1 models.LRPStartAuction
	}
	AddTaskForAuctionStub        func(models.Task)
	addTaskForAuctionMutex       sync.RWMutex
	addTaskForAuctionArgsForCall []struct {
		arg1 models.Task
	}
}

func (fake *FakeAuctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}{signals, ready})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		return fake.RunStub(signals, ready)
	} else {
		return fake.runReturns.result1
	}
}

func (fake *FakeAuctionRunner) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeAuctionRunner) RunArgsForCall(i int) (<-chan os.Signal, chan<- struct{}) {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.runArgsForCall[i].signals, fake.runArgsForCall[i].ready
}

func (fake *FakeAuctionRunner) RunReturns(result1 error) {
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeAuctionRunner) AddLRPStartAuction(arg1 models.LRPStartAuction) {
	fake.addLRPStartAuctionMutex.Lock()
	fake.addLRPStartAuctionArgsForCall = append(fake.addLRPStartAuctionArgsForCall, struct {
		arg1 models.LRPStartAuction
	}{arg1})
	fake.addLRPStartAuctionMutex.Unlock()
	if fake.AddLRPStartAuctionStub != nil {
		fake.AddLRPStartAuctionStub(arg1)
	}
}

func (fake *FakeAuctionRunner) AddLRPStartAuctionCallCount() int {
	fake.addLRPStartAuctionMutex.RLock()
	defer fake.addLRPStartAuctionMutex.RUnlock()
	return len(fake.addLRPStartAuctionArgsForCall)
}

func (fake *FakeAuctionRunner) AddLRPStartAuctionArgsForCall(i int) models.LRPStartAuction {
	fake.addLRPStartAuctionMutex.RLock()
	defer fake.addLRPStartAuctionMutex.RUnlock()
	return fake.addLRPStartAuctionArgsForCall[i].arg1
}

func (fake *FakeAuctionRunner) AddTaskForAuction(arg1 models.Task) {
	fake.addTaskForAuctionMutex.Lock()
	fake.addTaskForAuctionArgsForCall = append(fake.addTaskForAuctionArgsForCall, struct {
		arg1 models.Task
	}{arg1})
	fake.addTaskForAuctionMutex.Unlock()
	if fake.AddTaskForAuctionStub != nil {
		fake.AddTaskForAuctionStub(arg1)
	}
}

func (fake *FakeAuctionRunner) AddTaskForAuctionCallCount() int {
	fake.addTaskForAuctionMutex.RLock()
	defer fake.addTaskForAuctionMutex.RUnlock()
	return len(fake.addTaskForAuctionArgsForCall)
}

func (fake *FakeAuctionRunner) AddTaskForAuctionArgsForCall(i int) models.Task {
	fake.addTaskForAuctionMutex.RLock()
	defer fake.addTaskForAuctionMutex.RUnlock()
	return fake.addTaskForAuctionArgsForCall[i].arg1
}

var _ auctiontypes.AuctionRunner = new(FakeAuctionRunner)
