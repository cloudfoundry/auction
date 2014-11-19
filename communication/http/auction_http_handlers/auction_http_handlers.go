package auction_http_handlers

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func New(rep auctiontypes.AuctionRep, logger lager.Logger) rata.Handlers {
	handlers := rata.Handlers{
		routes.State:   &state{rep: rep, logger: logger},
		routes.Perform: &perform{rep: rep, logger: logger},

		routes.Sim_Reset: &reset{rep: rep, logger: logger},
	}

	return handlers
}
