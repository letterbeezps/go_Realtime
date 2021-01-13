package websocket

import (
	myUtil "Realtime/pkg/utils"
	"fmt"
)

type Pool struct {
	Register   chan *Client // 使用无缓冲通道
	Unregister chan *Client
	Clients    map[*Client]bool
	FDs        map[int]*Client // key: websocket连接的fd value: *CLient
	Broadcast  chan Message
	Push       chan Message // 服务端推送消息
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client, 5), // 当有新人加入时，通知
		Unregister: make(chan *Client, 5), // 处理有人退出
		Clients:    make(map[*Client]bool),
		FDs:        make(map[int]*Client),
		Broadcast:  make(chan Message, 5),
		Push:       make(chan Message, 5),
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			FD := myUtil.WebsocketFD(client.Conn)
			pool.Clients[client] = true
			fmt.Println("Size of Connection Pool: ", len(pool.Clients))
			for client, _ := range pool.Clients {
				fmt.Println(client)
				client.Conn.WriteJSON(Message{Fd: FD, Type: 1, Body: "New User Joined..."})
			}
			break
		case client := <-pool.Unregister:
			FD := myUtil.WebsocketFD(client.Conn)
			delete(pool.FDs, FD)
			delete(pool.Clients, client)
			fmt.Println("Size of Connection Pool: ", len(pool.Clients))
			for client, _ := range pool.Clients {
				client.Conn.WriteJSON(Message{Fd: FD, Type: 1, Body: "User Disconnected..."})
			}
			break
		case message := <-pool.Broadcast:
			fmt.Println("Sending message to all clients in Pool")
			for client, _ := range pool.Clients {
				if err := client.Conn.WriteJSON(message); err != nil {
					fmt.Println(err)
					return
				}
			}
			break
		case message := <-pool.Push:
			FD := message.Fd
			for fd, client := range pool.FDs {
				if fd != FD {
					if err := client.Conn.WriteJSON(message); err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}
	}
}
