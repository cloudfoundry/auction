package auctiontypes

import (
	"sort"

	"github.com/cloudfoundry-incubator/auction/util"
)

func (a StopScoreResults) Len() int           { return len(a) }
func (a StopScoreResults) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a StopScoreResults) Less(i, j int) bool { return a[i].Score < a[j].Score }

func (v StopScoreResults) AllFailed() bool {
	return len(v.FilterErrors()) == 0
}

func (v StopScoreResults) Reps() RepGuids {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.Rep)
	}

	return out
}

func (v StopScoreResults) InstanceGuids() []string {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.InstanceGuids...)
	}

	return out
}

func (v StopScoreResults) FilterErrors() StopScoreResults {
	out := StopScoreResults{}
	for _, r := range v {
		if r.Error == "" {
			out = append(out, r)
		}
	}

	return out
}

func (v StopScoreResults) Shuffle() StopScoreResults {
	out := make(StopScoreResults, len(v))

	perm := util.R.Perm(len(v))
	for i, index := range perm {
		out[i] = v[index]
	}

	return out
}

func (v StopScoreResults) Sort() StopScoreResults {
	sort.Sort(v)
	return v
}
