# Auction

**Note**: This repository should be imported as `code.cloudfoundry.org/auction`.

## Reporting issues and requesting features

Please report all issues and feature requests in [cloudfoundry/diego-release](https://github.com/cloudfoundry/diego-release/issues).

#### Learn more about Diego and its components at [diego-design-notes](https://github.com/cloudfoundry/diego-design-notes)

The `auction` package in this repository encodes the details behind Diego's scheduling mechanism.  There are two components in Diego that participate in auctions:

- The [Auctioneer](https://github.com/cloudfoundry/auctioneer) is responsible for holding auctions whenever a Task or LongRunningProcess needs to be scheduled.
- The [Rep](https://github.com/cloudfoundry/rep) represents a Diego Cell in the auction by making bids and, if picked as the winner, running the Task or LongRunningProcess.

The Auctioneers run on the Diego "Brain" nodes, and there is only ever one active Auctioneer at a time (determined by acquiring a lock in Locket). There is one Rep running on every Diego Cell.

The Auctioneer communicates with Reps on all Cells when holding an auction.

## The Auction Runner

The `auctionrunner` package provides an [*ifrit* process runner](https://github.com/tedsuo/ifrit/blob/master/runner.go) which consumes an incoming stream of requested auction work, batches it up, communicates with the Cell reps, picks winners, and then instructs the Cells to perform the work.

## The Simulation

The `simulation` package contains a Ginkgo test suite that describes a number of scheduling scenarios.  The `simulation` generates comprehensive output to the command line, and an SVG describing, visually, the results of the simulation run.

The simulation spins up a number of in-process [`SimulationRep`](https://github.com/cloudfoundry/auction/blob/master/simulation/simulationrep/simulation_rep.go)s.  They implement the [Rep client interface](https://github.com/cloudfoundry-incubator/rep/blob/master/client.go#L41-L54). This in-process communication mode allows us to isolate the algorithmic details from the communication details.  It allows us to iterate on the scoring math and scheduling details quickly and efficiently.

### Running on Diego

Instead of running the simulations by running `ginkgo` locally, you can run the Diego scheduling simulations on a Diego deployment itself!  See the [Diego Cluster Simulations repository](https://github.com/pivotal-cf-experimental/diego-cluster-simulations).
