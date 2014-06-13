package nats

import "reflect"

type Subjects struct {
	TotalResources              string
	Reset                       string
	SimulatedInstances          string
	SetSimulatedInstances       string
	BidForStartAuction          string
	BidForStopAuction           string
	RebidThenTentativelyReserve string
	ReleaseReservation          string
	Run                         string
	Stop                        string
}

func NewSubjects(repGuid string) Subjects {
	return Subjects{
		TotalResources:              repGuid + ".total-resources",
		Reset:                       repGuid + ".reset",
		SimulatedInstances:          repGuid + ".simulated-instances",
		SetSimulatedInstances:       repGuid + ".set-simulated-instances",
		BidForStartAuction:          repGuid + ".bid-for-start-auction",
		BidForStopAuction:           repGuid + ".bid-for-stop-auction",
		RebidThenTentativelyReserve: repGuid + ".rebid-then-tentatively-reserve",
		ReleaseReservation:          repGuid + ".release-reservation",
		Run:                         repGuid + ".run",
		Stop:                        repGuid + ".stop",
	}
}

func (subjects Subjects) Slice() []string {
	val := reflect.ValueOf(subjects)
	count := val.NumField()
	out := make([]string, count)
	for i := 0; i < count; i++ {
		out[i] = val.Field(i).String()
	}
	return out
}
