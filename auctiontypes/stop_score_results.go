package auctiontypes

import (
	"sort"

	"github.com/cloudfoundry-incubator/auction/util"
)

func (a StopAuctionBids) Len() int           { return len(a) }
func (a StopAuctionBids) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StopAuctionBids) Less(i, j int) bool { return a[i].Bid < a[j].Bid }

func (v StopAuctionBids) AllFailed() bool {
	return len(v.FilterErrors()) == 0
}

func (v StopAuctionBids) Reps() RepGuids {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.Rep)
	}

	return out
}

func (v StopAuctionBids) InstanceGuids() []string {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.InstanceGuids...)
	}

	return out
}

func (v StopAuctionBids) FilterErrors() StopAuctionBids {
	out := StopAuctionBids{}
	for _, r := range v {
		if r.Error == "" {
			out = append(out, r)
		}
	}

	return out
}

func (v StopAuctionBids) Shuffle() StopAuctionBids {
	out := make(StopAuctionBids, len(v))

	perm := util.R.Perm(len(v))
	for i, index := range perm {
		out[i] = v[index]
	}

	return out
}

func (v StopAuctionBids) Sort() StopAuctionBids {
	sort.Sort(v)
	return v
}
