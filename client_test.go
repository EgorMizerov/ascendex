package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

const BBOResponse = `{
    "m": "bbo",
    "symbol": "BTC/USDT",
    "data": {
        "ts": 1573068442532,
        "bid": [
            "9309.11",
            "0.0197172"
        ],
        "ask": [
            "9309.12",
            "0.8851266"
        ]
    }
}`

func setupReadMessages(t *testing.T, sut *AscendexClient) {
	err := sut.Connection()
	assert.NoError(t, err)
	err = sut.SubscribeToChannel("test")
	assert.NoError(t, err)
}

func TestAscendexClient_ReadMessagesFromChannel(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(websocketHandler))
	defer server.Close()
	sut := NewAscendexClient("ws" + strings.TrimPrefix(server.URL, "http"))
	setupReadMessages(t, sut)
	resultChan := make(chan BestOrderBook, 2)

	// Act
	go sut.ReadMessagesFromChannel(resultChan)
	result := <-resultChan

	// Assert
	assert.Equal(t, BestOrderBook{
		Ask: Order{
			Amount: 0.8851266,
			Price:  9309.12,
		},
		Bid: Order{
			Amount: 0.0197172,
			Price:  9309.11,
		},
	}, result)
}

func TestAscendexClient_Connection(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(websocketHandler))
	defer server.Close()
	sut := NewAscendexClient("ws" + strings.TrimPrefix(server.URL, "http"))

	err := sut.Connection()

	assert.NoError(t, err)
	assert.NotNil(t, sut.conn)
}

func TestAscendexClient_Connection_FailedConnect(t *testing.T) {
	// Arrange
	sut := NewAscendexClient("ws" + strings.TrimPrefix("localhost:8080", "http"))

	err := sut.Connection()

	assert.Error(t, err)
	assert.Nil(t, sut.conn)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	for {
		_, response, err := c.ReadMessage()
		if err != nil {
			break
		}
		message := gjson.Parse(string(response))
		switch message.Get("op").String() {
		case "ping":
			err = c.WriteMessage(websocket.TextMessage, []byte(`{"m": "pong"}`))
			if err != nil {
				break
			}
		case "sub":
			go func() {
				for {
					err = c.WriteMessage(websocket.TextMessage, []byte(BBOResponse))
					if err != nil {
						break
					}
					time.Sleep(time.Second * 6)
				}
			}()
		}
	}
}
