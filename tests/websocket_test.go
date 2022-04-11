package tests

import (
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	centrifuge2 "bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"bufio"
	"encoding/json"
	"github.com/centrifugal/centrifuge-go"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestCentrifuge(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe(":5000", nil))
	}()

	handler := centrifuge2.NewEventHandler()

	c := centrifuge2.NewClient(handler, infra.CentrifugeConfig{
		Host: "localhost:8000",
	})

	defer func() { _ = c.Close() }()

	sub, err := c.NewSubscription("candle_chart_BTC_1m")
	if err != nil {
		log.Fatalln(err)
	}

	sub.OnPublish(handler)
	sub.OnSubscribeSuccess(handler)
	sub.OnSubscribeError(handler)
	sub.OnUnsubscribe(handler)

	err = sub.Subscribe()
	if err != nil {
		log.Fatalln(err)
	}

	pubText := func(text string) error {
		msg := &domain.Candle{
			Open:      12,
			High:      34,
			Low:       8,
			Close:     41,
			Volume:    0,
			Timestamp: time.Time{},
		}
		data, _ := json.Marshal(msg)
		_, err := sub.Publish(data)
		return err
	}

	err = c.Connect()
	if err != nil {
		log.Fatalln(err)
	}

	err = pubText("hello")
	if err != nil {
		log.Printf("Error publish: %s", err)
	}

	go func(sub *centrifuge.Subscription) {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			println(text)
			//err := pubText(text)
			/*if err != nil {
				log.Printf("Error publish: %s", err)
			}*/
		}
	}(sub)

	time.Sleep(3 * time.Second)
}