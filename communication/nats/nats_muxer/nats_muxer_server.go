package nats_muxer

import (
	"encoding/json"

	"github.com/apcera/nats"
	"github.com/cloudfoundry/yagnats"
)

type MuxedHandler func([]byte) []byte

func HandleMuxedNATSRequest(client yagnats.NATSConn, subject string, callback MuxedHandler) (*nats.Subscription, error) {
	return client.Subscribe(subject, func(msg *nats.Msg) {
		request := message{}
		err := json.Unmarshal(msg.Data, &request)
		if err != nil {
			return
		}

		payload := callback(request.Payload)

		response := message{
			CorrelationID: request.CorrelationID,
			Payload:       payload,
		}

		responsePayload, err := json.Marshal(response)
		if err != nil {
			return
		}

		client.Publish(msg.Reply, responsePayload)
	})
}
