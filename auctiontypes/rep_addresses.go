package auctiontypes

import (
	"math"

	"github.com/cloudfoundry-incubator/auction/util"
)

func (r RepAddresses) RandomSubsetByCount(n int) RepAddresses {
	if len(r) < n {
		return r
	}

	permutation := util.R.Perm(len(r))
	subset := make(RepAddresses, n)
	for i, index := range permutation[:n] {
		subset[i] = r[index]
	}

	return subset
}

func (r RepAddresses) RandomSubsetByFraction(f float64, minNumber int) RepAddresses {
	if f >= 1 {
		return r
	}

	n := int(math.Ceil(float64(len(r)) * f))
	if n < minNumber {
		n = minNumber
	}

	return r.RandomSubsetByCount(n)
}

func (r RepAddresses) Without(repGuids ...string) RepAddresses {
	lookup := map[string]bool{}
	for _, repGuid := range repGuids {
		lookup[repGuid] = true
	}

	out := RepAddresses{}
	for _, repAddress := range r {
		if !lookup[repAddress.RepGuid] {
			out = append(out, repAddress)
		}
	}

	return out
}

func (r RepAddresses) Lookup(repGuids ...string) RepAddresses {
	lookup := map[string]bool{}
	for _, repGuid := range repGuids {
		lookup[repGuid] = true
	}

	out := RepAddresses{}
	for _, repAddress := range r {
		if lookup[repAddress.RepGuid] {
			out = append(out, repAddress)
		}
	}

	return out
}

func (r RepAddresses) AddressFor(repGuid string) RepAddress {
	for _, repAddress := range r {
		if repAddress.RepGuid == repGuid {
			return repAddress
		}
	}

	return RepAddress{}
}
