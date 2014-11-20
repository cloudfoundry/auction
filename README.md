[![Build Status](https://travis-ci.org/cloudfoundry-incubator/auction.svg)](https://travis-ci.org/cloudfoundry-incubator/auction)

# Auction

####Learn more about Diego and its components at [diego-design-notes](https://github.com/cloudfoundry-incubator/diego-design-notes)

The `auction` package in this repository encodes the details behind Diego's scheduling mechanism.  There are two components in Diego that participate in auctions:

- The [Auctioneer](https://github.com/cloudfoundry-incubator/auctioneer) is responsible for holding auctions whenever a LongRunningProcess needs to be started.
- The [Rep](https://github.com/cloudfoundry-incubator/rep) represents a Diego Cell in the auction by making bids and, if picked as the winner, running the LongRunningProcess.

There is one Auctioneer and and one Rep on each Diego Cell.  All auctions are held by one Auctioneer with the others standing by as backups in case of failure.

The Auctioneer communicates with Reps on all other Cells when holding an auction.

## Subpackages and Usage

There are a number of subpackages to the auction:

- `auctionrunner`: The auctionrunner consumes an incoming stream of requested auction work, batches it up, communicates with the Cell reps, picks winners, and then instructs the Cells to perform the work.

- `communication/http`: Provides an `http` based communication layer.
    - `communication/http/auction_http_client` provides an `auctiontypes.CellRep` used by Auctioneers to communicate with Reps over http.
    - `communication/http/auction_http_handlers` provides a set of http handlers.  Reps participates in an http-based auction by running an http server that mounts these endpoints.

## The Simulation

The `simulation` package contains a Ginkgo test suite that describes a number of scheduling scenarios.  These scenarios can be run in a number of different modes, all controlled by passing flags to the test suite.  The `simulation` generates comprehensive output to the command line, and an SVG describing, visually, the results of the simulation run.

### In-Process Communication

By default, the simulation runs with an "in-process" communication model.  In this mode, the simulation spins up a number of in-process `CellReps`.  The `CellReps` implement a minimal, simple, in-memory implementation of the `auctiontypes.CellRep` interface.

This in-process communication mode allows us to isolate the algorithmic details from the communication details.  It allows us to iterate on the scoring math and scheduling details quickly and efficiently.

### NATS Communication

The in-process model outlined above provides us with a starting point for analyzing the auction.  To understand the impact of http communication, and ensure the http layer works correctly, we can run the simulation with `ginkgo -- --communicationMode=http`.

When `communicationMode` is set to `http`, the simulation will spin up 100 `simulation/repnode` external processes.   The simulation then runs in-process auctions that communicate with these external processes via http.

### Running on Diego

github.com/pivotal-cf-experimental/diego-cluster-simulations has a simulation suite that runs against diego.