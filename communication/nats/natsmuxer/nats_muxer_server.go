package natsmuxer

import (
	"encoding/json"

	"github.com/cloudfoundry/yagnats"
)

type MuxedHandler func([]byte) []byte

func HandleMuxedNATSRequest(client yagnats.NATSClient, subject string, callback MuxedHandler) (int64, error) {
	return client.Subscribe(subject, func(msg *yagnats.Message) {
		request := message{}
		err := json.Unmarshal(msg.Payload, &request)
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

		client.Publish(msg.ReplyTo, responsePayload)
	})
}
