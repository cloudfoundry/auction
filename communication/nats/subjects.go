package nats

import "reflect"

type Subjects struct {
	TotalResources              string
	Reset                       string
	SimulatedInstances          string
	SetSimulatedInstances       string
	Score                       string
	StopScore                   string
	ScoreThenTentativelyReserve string
	ReleaseReservation          string
	Run                         string
	Stop                        string
}

func NewSubjects(guid string) Subjects {
	return Subjects{
		TotalResources:        guid + ".total-resources",
		Reset:                 guid + ".reset",
		SimulatedInstances:    guid + ".simulated-instances",
		SetSimulatedInstances: guid + ".set-simulated-instances",
		Score:                       guid + ".score",
		StopScore:                   guid + ".stop-score",
		ScoreThenTentativelyReserve: guid + ".score-then-tentatively-reserve",
		ReleaseReservation:          guid + ".release-reservation",
		Run:                         guid + ".run",
		Stop:                        guid + ".stop",
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
