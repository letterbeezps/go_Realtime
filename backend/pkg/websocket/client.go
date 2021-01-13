package websocket

import (
	"fmt"
	"log"

	myUtil "Realtime/pkg/utils"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
}

type Message struct {
	Fd   int    `json:"fd"`
	Type int    `json:"type"`
	Body string `json:"body"`
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		fmt.Printf("等待中")
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		FD := myUtil.WebsocketFD(c.Conn)

		message := Message{Fd: FD, Type: messageType, Body: string(p)}
		// c.Pool.Push <- message
		// c.Pool.Broadcast <- message
		if string(p) == "Happy" {
			c.Pool.Push <- message
		} else {
			c.Pool.Broadcast <- message
		}

		fmt.Printf("Message Received: %+v\n", message)
	}
}
