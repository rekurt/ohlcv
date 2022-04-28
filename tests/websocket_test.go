package tests

import (
	"bufio"
	"encoding/json"

	cfge "github.com/centrifugal/centrifuge-go"

	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	centrifuge2 "bitbucket.org/novatechnologies/ohlcv/infra/centrifugo"

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

	handler := centrifuge2.NewLoggingEventHandler()

	c, err := centrifuge2.NewWSClient(
		infra.CentrifugoClientConfig{
			Addr: "localhost:8000",
		},
	)

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
			Open:      domain.MustParseDecimal("12"),
			High:      domain.MustParseDecimal("34"),
			Low:       domain.MustParseDecimal("8"),
			Close:     domain.MustParseDecimal("41"),
			Volume:    domain.MustParseDecimal("0"),
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

	go func(sub *cfge.Subscription) {
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
