package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type AscendexClient struct {
	wsURL string
	conn  *websocket.Conn
}

func NewAscendexClient(wsURL string) *AscendexClient {
	return &AscendexClient{
		wsURL: wsURL,
	}
}

func (self *AscendexClient) Connection() error {
	conn, _, err := websocket.DefaultDialer.Dial(self.wsURL, nil)
	if err != nil {
		return err
	}
	self.conn = conn
	return nil
}

func (self *AscendexClient) Disconnect() {
	self.conn.Close()
}

type SubscribeToChannelRequest struct {
	Op string `json:"op"`
	Ch string `json:"ch"`
}

func (self *AscendexClient) SubscribeToChannel(symbol string) error {
	if self.conn == nil {
		return errors.New("no connection to websocket channel")
	}
	if err := self.conn.WriteJSON(SubscribeToChannelRequest{
		Op: "sub",
		Ch: fmt.Sprintf("bbo:%s", self.convertSymbol(symbol)),
	}); err != nil {
		self.Disconnect()
		return err
	}

	return nil
}

func (self *AscendexClient) convertSymbol(source string) string {
	result := strings.Split(source, "_")
	if len(result) < 2 {
		return ""
	}
	return fmt.Sprintf("%s/%s", result[0], result[1])
}

func (self *AscendexClient) ReadMessagesFromChannel(ch chan<- BestOrderBook) {
	if self.conn == nil {
		return
	}

	for {
		_, response, err := self.conn.ReadMessage()
		if err != nil {
			close(ch)
			return
		}

		message := gjson.Parse(string(response))
		topic := message.Get("m").String()

		if topic == "bbo" {
			var result = BestOrderBook{
				Bid: Order{
					Price:  message.Get("data.bid.0").Float(),
					Amount: message.Get("data.bid.1").Float(),
				},
				Ask: Order{
					Price:  message.Get("data.ask.0").Float(),
					Amount: message.Get("data.ask.1").Float(),
				},
			}
			ch <- result
		}
	}
}

func (self *AscendexClient) WriteMessagesToChannel() {
	if self.conn == nil {
		return
	}

	for {
		if err := self.conn.WriteMessage(websocket.TextMessage, []byte(`{"op": "ping"}`)); err != nil {
			return
		}
		time.Sleep(time.Second * 15)
	}
}
