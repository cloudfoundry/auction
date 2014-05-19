package types

import (
	"errors"
	"time"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type AuctionRequest struct {
	Instance Instance     `json:"i"`
	RepGuids RepGuids     `json:"rg"`
	Rules    AuctionRules `json:"r"`
}

type AuctionResult struct {
	Instance          Instance      `json:"i"`
	Winner            string        `json:"w"`
	NumRounds         int           `json:"nr"`
	NumCommunications int           `json:"nc"`
	BiddingDuration   time.Duration `json:"bd"`
	Duration          time.Duration `json:"d"`
}

type AuctionRules struct {
	Algorithm      string  `json:"alg"`
	MaxRounds      int     `json:"mr"`
	MaxBiddingPool float64 `json:"mb"`
}

type RepGuids []string

type ScoreResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type ScoreResults []ScoreResult

type Resources struct {
	DiskMB     float64 `json:"d"`
	MemoryMB   float64 `json:"m"`
	Containers int     `json:"c,omitempty"`
}

type Instance struct {
	AppGuid      string    `json:"a"`
	InstanceGuid string    `json:"i"`
	Resources    Resources `json:"r"`
}

type RepPoolClient interface {
	Score(guids []string, instance Instance) ScoreResults
	ScoreThenTentativelyReserve(guids []string, instance Instance) ScoreResults
	ReleaseReservation(guids []string, instance Instance)
	Claim(guid string, instance Instance)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) Resources
	Instances(guid string) []Instance
	SetInstances(guid string, instances []Instance)
	Reset(guid string)
}
