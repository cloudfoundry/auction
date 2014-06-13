package nats

import "reflect"

type Subjects struct {
	TotalResources              string
	Reset                       string
	SimulatedInstances          string
	SetSimulatedInstances       string
	Score                       string
	ScoreThenTentativelyReserve string
	ReleaseReservation          string
	Run                         string
}

func NewSubjects(guid string) Subjects {
	return Subjects{
		TotalResources:        guid + ".total_resources",
		Reset:                 guid + ".reset",
		SimulatedInstances:    guid + ".simulated_instances",
		SetSimulatedInstances: guid + ".set_simulated_instances",
		Score: guid + ".score",
		ScoreThenTentativelyReserve: guid + ".score_then_tentatively_reserve",
		ReleaseReservation:          guid + ".release-reservation",
		Run:                         guid + ".run",
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
