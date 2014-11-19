package routes

import "github.com/tedsuo/rata"

const (
	State   = "STATE"
	Perform = "PERFORM"

	Sim_Reset = "RESET"
)

var Routes = rata.Routes{
	{Path: "/state", Method: "GET", Name: State},
	{Path: "/work", Method: "POST", Name: Perform},

	{Path: "/sim/reset", Method: "POST", Name: Sim_Reset},
}
