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

func (r RepGuids) RandomSubsetByFraction(f float64, minNumber int) RepGuids {
	if f >= 1 {
		return r
	}

	n := int(math.Ceil(float64(len(r)) * f))
	if n < minNumber {
		n = minNumber
	}

	return r.RandomSubsetByCount(n)
}

func (r RepGuids) Without(repGuids ...string) RepGuids {
	lookup := map[string]bool{}
	for _, repGuid := range repGuids {
		lookup[repGuid] = true
	}

	out := RepGuids{}
	for _, repGuid := range r {
		if !lookup[repGuid] {
			out = append(out, repGuid)
		}
	}

	return out
}
