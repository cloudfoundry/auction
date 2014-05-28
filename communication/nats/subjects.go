package nats

import "reflect"

type Subjects struct {
	TotalResources              string
	Reset                       string
	LrpAuctionInfos             string
	SetLrpAuctionInfos          string
	Score                       string
	ScoreThenTentativelyReserve string
	ReleaseReservation          string
	Run                         string
}

func NewSubjects(guid string) Subjects {
	return Subjects{
		TotalResources:     guid + ".total_resources",
		Reset:              guid + ".reset",
		LrpAuctionInfos:    guid + ".lrp_auction_infos",
		SetLrpAuctionInfos: guid + ".set_lrp_auction_infos",
		Score:              guid + ".score",
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
