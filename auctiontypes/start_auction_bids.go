package auctiontypes

import (
	"sort"

	"github.com/cloudfoundry-incubator/auction/util"
)

func (a StartAuctionBids) Len() int           { return len(a) }
func (a StartAuctionBids) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StartAuctionBids) Less(i, j int) bool { return a[i].Bid < a[j].Bid }

func (v StartAuctionBids) AllFailed() bool {
	return len(v.FilterErrors()) == 0
}

func (v StartAuctionBids) Reps() []string {
	out := []string{}
	for _, r := range v {
		out = append(out, r.Rep)
	}

	return out
}

func (v StartAuctionBids) FilterErrors() StartAuctionBids {
	out := StartAuctionBids{}
	for _, r := range v {
		if r.Error == "" {
			out = append(out, r)
		}
	}

	return out
}

func (v StartAuctionBids) Shuffle() StartAuctionBids {
	out := make(StartAuctionBids, len(v))

	perm := util.R.Perm(len(v))
	for i, index := range perm {
		out[i] = v[index]
	}

	return out
}

func (v StartAuctionBids) Sort() StartAuctionBids {
	sort.Sort(v)
	return v
}
