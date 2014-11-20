[![Build Status](https://travis-ci.org/cloudfoundry-incubator/auction.svg)](https://travis-ci.org/cloudfoundry-incubator/auction)

# Auction

####Learn more about Diego and its components at [diego-design-notes](https://github.com/cloudfoundry-incubator/diego-design-notes)

The `auction` package in this repository encodes the details behind Diego's distributed auction scheduling mechanism.  There are two components in Diego that participate distributed auctions:

- The [Auctioneer](https://github.com/cloudfoundry-incubator/auctioneer) is responsible for holding auctions whenever a LongRunningProcess needs to be started.
- The [Rep](https://github.com/cloudfoundry-incubator/rep) represents a Diego Cell in the auction by making bids and, if picked as the winner, running the LongRunningProcess.

There is one Auctioneer and and one Rep on each Diego Cell.  The Auctioneer on a given Cell may communicate with Reps on all other Cells when holding an auction.

There are two principal, and separable, components to the auction: the first is the underlying auction algorithm, the second is the communication medium used to mediate communication between Auctioneers and Reps.

The first of these (the auction algorithm) can be broken into two concerns: the algorithm used by the Auctioneer to ask for bids and select a winner, and the algoritm used by the Rep to compute scores.

All of these concerns are complex and interrelated.  The `auction` package exists to capture that complexity in one place.  In lieu of a traditional integration test, the `auction` includes a `simulation` subpackage that can excercise the auction in a variety of contexts and scenarios.  This simulation is used to rapidly iterate on the auction and validate the the underlying auction algorithm and communication layers work in real-life usecases.

## Subpackages and Usage

There are a number of subpackages to the auction:

### Subpackages Related to the Algorithm

- `auctionrep`: Provides an implementation of `auctiontypes.CellRep`.  This package implements the role of the Rep in an auction and can do things like compute a bid.  In order to do its work, the `CellRep` must be provided an implementation of `auctiontypes.CellRepDelegate`.  Diego's Rep implements this interface, passing in the delegate to the `CellRep`.  In this way the Rep has a clear and simple contract that it must satisfy and knows very little about the details of the auction.

- `auctionrunner`: Knows how to run auctions using a variety of different algorithms.  The AuctionRunner package must be passed an `auctiontypes.RepPoolClient` in order to communicate with the Reps.   The Auctioneer uses the `auctionrunner` package to actually run an auction and, therefore, knows very little about the details of the auction.

- `autionrunner/algorithms`: Implements a number of different auction algorithms.  These are described in the files under the algorithms package.

### Subpackages Related to Communication

- `communication/http`: Provides an `http` based communication layer.
    - `communication/http/auction_http_client` provides an `auctiontypes.RepPoolClient` used by Auctioneers to communicate with Reps over http.
    - `communication/http/auction_http_handlers` provides a set of http handlers.  Reps participates in an http-based auction by running an http server that mounts these endpoints.

- `communication/nats`: Provides a `nats` based communication layer.
    - `communication/nats/auction_nats_client` provides an `auctiontypes.RepPoolClient` used by Auctioneers to communicate with Reps over nats.
    - `communication/nats/auction_nats_server` provides a nats "server" (a series of nats subscriptions).  Reps participate in a nats-based auction by running this server.

There are important topological differences between the nats and http communication layers.  The nats communication model requires all communication to flow through a nats cluster.  As `N`, the number of Cells, grows the amount of traffic flowing through the nats cluster scales as `N^2`.  This is undesirable.

The http communication model comes with more overhead (http is a heavier-weight protocol than nats).  It, however, has the favorable property of being peer-to-peer.  With a peer-to-peer topology the amount of traffic flowing through any givne node scales as `N`.  This is quite desirable and, eventually, outweighs the overhead associated with http.

It's important to understand that Diego auctions are *demand-driven*.  When new work must be scheduled the Auctioneer holds an auction.  It does this by *asking* Reps for bids.  This is different from a supply-driven scheduling approach like Mesos where the "Reps" (nodes that have capacity to run things) reach out periodically with resource offers to a central coordinator (a node that keeps track of what needs to be run).

### How the Subpackages are Used

In order to hold an auction, the Auctioneer must:

- Create an `auctiontypes.RepPoolClient` (either `communication/http/auction_http_client` or `communication/http/auction_nats_client`).
- Create an `auctionrunner.AuctionRunner`, passing it the `auctiontypes.RepPoolClient`
- Call `auctionrunner.AuctionRunner.RunLRPStartAuction(...)` to hold an auction.

In order to participate in auctions, the Rep must:

- Provide a delegate satisfying `auctiontypes.CellRepDelegate`
- Pass said delegate to an instance of `auctionrep.CellRep`
- Spin up a server, passing in the constructed `auctionrep.CellRep`.  This will be either an http server with the `communication/http/auction_http_handlers` handlers, or the nats "server" provided by `communication/nats/auction_nats_server`.

## The Simulation

The `simulation` package contains a Ginkgo test suite that describes a number of scheduling scenarios.  These scenarios can be run in a number of different modes, all controlled by passing flags to the test suite.  The `simulation` generates comprehensive output to the command line, and an SVG describing, visually, the results of the simulation run.

### In-Process Communication

By default, the simulation runs with an "in-process" communication model.  In this mode, the simulation spins up a number of in-process `CellReps` and `AuctionRunners`.  The `CellReps` are provided a `simulationrepdelegate` that provides a minimal, simple, in-memory implementation of the `auctiontypes.CellRepDelegate` interface.

These in-process `AuctionRunners` "communicate" with the in-process `CellReps` via the `auctiontypes.RepPoolClient` provided by the `simulation/communication/inprocess` subpackage.

This in-process communication mode allows us to isolate the algorithmic details from the communication details.  It allows us to iterate on the scoring math and scheduling details quickly and efficiently.

### NATS and HTTP Communication

The in-process model outlined above provides us with a starting point for analyzing the auction.  The introduction of a latent communication medium adds new, complex, behavior to the auction.  We can explore this behavior by running the simulation with `ginkgo -- --communicationMode=MODE` where `MODE` is one of `inprocess` (the default), `nats`, and `http`.

When `communicationMode` is set to `nats` or `http`, the simulation will spin up 100 `simulation/local/repnode` external processes.  These processes run an `CellRep` with a `simulationrepdelegate` that listens on NATS fro the `nats` case and HTTP for the `http` case.  The simulation then runs in-process auctions that communicate with these external processes via the communication medium chosen.

It is difficult to reason about the performance implications of these communication schemes on a single computer -- the NATS and HTTP simulations are primarily intended to ensure the correctness of the communication implementation, not to measure its performance.  For that, one should run the simulation on a cluster (next section).

### Running on Diego

github.com/pivotal-cf-experimental/diego-cluster-simulations has a simulation suite that runs against diego.