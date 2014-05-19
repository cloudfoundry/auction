package auctiontypes

import (
	"math"

	"github.com/cloudfoundry-incubator/auction/util"
)

func (r RepGuids) RandomSubsetByCount(n int) RepGuids {
	if len(r) < n {
		return r
	}

	permutation := util.R.Perm(len(r))
	subset := make(RepGuids, n)
	for i, index := range permutation[:n] {
		subset[i] = r[index]
	}

	return subset
}

func (r RepGuids) RandomSubsetByFraction(f float64) RepGuids {
	if f >= 1 {
		return r
	}

	n := int(math.Ceil(float64(len(r)) * f))

	return r.RandomSubsetByCount(n)
}

func (r RepGuids) Without(guids ...string) RepGuids {
	lookup := map[string]bool{}
	for _, guid := range guids {
		lookup[guid] = true
	}

	out := RepGuids{}
	for _, guid := range r {
		if !lookup[guid] {
			out = append(out, guid)
		}
	}

	return out
}
