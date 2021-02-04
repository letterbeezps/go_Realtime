package epoll

import "github.com/gorilla/websocket"

type Epoller interface {
	Add(conn *websocket.Conn) error
	Remove(conn *websocket.Conn) error
	Wait(count int) ([]*websocket.Conn, error)
}
