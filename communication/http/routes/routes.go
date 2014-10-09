package routes

import "github.com/tedsuo/rata"

const (
	BidForStartAuction          = "BID_FOR_START_AUCTION"
	BidForStopAuction           = "BID_FOR_STOP_AUCTION"
	RebidThenTentativelyReserve = "REBID_THEN_TENTATIVELY_RESERVE"
	ReleaseReservation          = "RELEASE_RESERVATION"
	Run                         = "RUN"
	Stop                        = "STOP"

	Sim_TotalResources        = "TOTAL_RESOURCES"
	Sim_Reset                 = "RESET"
	Sim_SetSimulatedInstances = "SET_SIMULATED_INSTANCES"
	Sim_SimulatedInstances    = "SIMULATED_INSTANCES"
)

var Routes = rata.Routes{
	{Path: "/bids/start_auction", Method: "GET", Name: BidForStartAuction},
	{Path: "/bids/stop_auction", Method: "GET", Name: BidForStopAuction},

	{Path: "/reservations", Method: "POST", Name: RebidThenTentativelyReserve},
	{Path: "/reservations", Method: "DELETE", Name: ReleaseReservation},

	{Path: "/run", Method: "POST", Name: Run},
	{Path: "/stop", Method: "POST", Name: Stop},

	{Path: "/sim/total_resources", Method: "GET", Name: Sim_TotalResources},
	{Path: "/sim/reset", Method: "POST", Name: Sim_Reset},
	{Path: "/sim/simulated_instances", Method: "GET", Name: Sim_SimulatedInstances},
	{Path: "/sim/simulated_instances", Method: "POST", Name: Sim_SetSimulatedInstances},
}
