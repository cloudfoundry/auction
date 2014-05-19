package types

import (
	"sort"

	"github.com/onsi/auction/util"
)

func (a ScoreResults) Len() int           { return len(a) }
func (a ScoreResults) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScoreResults) Less(i, j int) bool { return a[i].Score < a[j].Score }

func (v ScoreResults) AllFailed() bool {
	return len(v.FilterErrors()) == 0
}

func (v ScoreResults) Reps() RepGuids {
	out := RepGuids{}
	for _, r := range v {
		out = append(out, r.Rep)
	}

	return out
}

func (v ScoreResults) FilterErrors() ScoreResults {
	out := ScoreResults{}
	for _, r := range v {
		if r.Error == "" {
			out = append(out, r)
		}
	}

	return out
}

func (v ScoreResults) Shuffle() ScoreResults {
	out := make(ScoreResults, len(v))

	perm := util.R.Perm(len(v))
	for i, index := range perm {
		out[i] = v[index]
	}

	return out
}

func (v ScoreResults) Sort() ScoreResults {
	sort.Sort(v)
	return v
}
