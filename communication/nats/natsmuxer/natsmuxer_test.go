package natsmuxer_test

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	. "github.com/cloudfoundry-incubator/auction/communication/nats/natsmuxer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nats Muxer", func() {
	var subscriptionID1, subscriptionID2 int64
	var client *NATSMuxerClient

	BeforeEach(func() {
		var err error

		client = NewNATSMuxerClient(natsClient)
		err = client.ListenForResponses()
		Ω(err).ShouldNot(HaveOccurred())

		subscriptionID1, err = HandleMuxedNATSRequest(natsClient, "echo", func(payload []byte) []byte {
			return payload
		})
		Ω(err).ShouldNot(HaveOccurred())

		subscriptionID2, err = HandleMuxedNATSRequest(natsClient, "square", func(payload []byte) []byte {
			i, err := strconv.Atoi(string(payload))
			Ω(err).ShouldNot(HaveOccurred())

			return []byte(strconv.Itoa(i * i))
		})

		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		err := natsClient.Unsubscribe(subscriptionID1)
		Ω(err).ShouldNot(HaveOccurred())

		err = natsClient.Unsubscribe(subscriptionID2)
		Ω(err).ShouldNot(HaveOccurred())

		err = client.Shutdown()
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("should be able to correctly route requests/responses", func() {
		wg := &sync.WaitGroup{}

		for i := 1; i < 20; i++ {
			wg.Add(1)
			i := i
			go func() {
				response, err := client.Request("square", []byte(strconv.Itoa(i)), time.Second)
				Ω(err).ShouldNot(HaveOccurred())

				result, err := strconv.Atoi(string(response))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(result).Should(Equal(i * i))
				wg.Done()
			}()
		}

		for i := 1; i < 20; i++ {
			wg.Add(1)
			i := i
			go func() {
				message := []byte(fmt.Sprintf("hello world %d", i))
				response, err := client.Request("echo", message, time.Second)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(response).Should(Equal(message))

				wg.Done()
			}()
		}

		wg.Wait()
	})

	It("should be able to timeout", func() {
		response, err := client.Request("foo", []byte("foo"), time.Second)
		Ω(err).Should(MatchError(TimeoutError))
		Ω(response).Should(BeEmpty())
	})
})
