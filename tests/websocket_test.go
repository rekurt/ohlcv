package tests

import (
	"bitbucket.org/novatechnologies/ohlcv/domain"
	centrifuge2 "bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"
	"bufio"
	"encoding/json"
	"flag"
	"github.com/centrifugal/centrifuge-go"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var addr = flag.String("addr", "127.0.01:8082", "http service address")

/*func TestWebsocketClient(t *testing.T) {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws/candles"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}*/

func TestCentrifuge(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe(":5000", nil))
	}()

	handler := centrifuge2.NewEventHandler()

	c := centrifuge2.NewClient(handler)
	defer func() { _ = c.Close() }()

	sub, err := c.NewSubscription("candle_chart:$market_$interval")
	if err != nil {
		log.Fatalln(err)
	}

	sub.OnPublish(handler)
	sub.OnJoin(handler)
	sub.OnLeave(handler)
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

	log.Printf("Print something and press ENTER to send\n")
	go func(sub *centrifuge.Subscription) {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			err := pubText(text)
			if err != nil {
				log.Printf("Error publish: %s", err)
			}
		}
	}(sub)
}