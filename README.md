[![Build Status](https://travis-ci.org/cloudfoundry-incubator/auction.svg)](https://travis-ci.org/cloudfoundry-incubator/auction)

# Auction

####Learn more about Diego and its components at [diego-design-notes](https://github.com/cloudfoundry-incubator/diego-design-notes)

Auction implements the scheduling algorithm for Diego's long running processes.  It does this by codifying two players in the scheduling game:

## The Auctioneer

The `auctioneer` package provides a variety of auction algorithms.  Diego nodes that play the `auctioneer` role must call `auctioneer.Auction` passing in a valid `auctiontypes.StartAuctionRequest` and `auctiontypes.RepPoolClient` for communciating with the pool of auction representatives.

## The Representatives

The `auctionrep` package provides an implementation of `AuctionRep`.  These `AuctionRep`s follow the rules of the auction correctly but need to be provided with an `AuctionRepDelegate` that performs the actual work of tracking resources, reserving instances, and starting them running.

## Communication

The auctioneers must be able to communicate with the auctionreps via some protocol.  The communication package provides implementations for `servers` (to be run on the representative nodes) and `clients` to be constructed and used on the `auctioneer` node.

Currently `Auction` provides one remote communication packages: `nats`.

## Simulation

Because communication has been separated from implementation, and because the implementation of the auctioneer and auctionrep has been built to be reusable, it is possible to construct a comprehensive simulation to test the various scheduling algorithms, using various communication schemes, on various infrastructures.

This is done in the simulation package which is the defacto "test suite" that ensures the auction is played correctly.  As new scheduling features are added, a corresponding simulation should be added to the simulation suite.

In addition to `nats`, the simulation suite provides an *inprocess* means of communication.  This allows a feel of representatives and auctioneers to be started as goroutines in-process and allows for rapid iteration on the underlying scheduling algorithm.
