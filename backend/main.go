package main

import (
	"fmt"
	"log"
	"net/http"

	myEpoll "Realtime/pkg/epoll"
	myUtil "Realtime/pkg/utils"
	myWS "Realtime/pkg/websocket"

	"github.com/gin-gonic/gin"
)

var epoller myEpoll.Epoller

func serveWs(pool *myWS.Pool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("WedSocket EndPoint Hit")

	conn, err := myWS.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	} else {
		log.Println("成功建立连接")
	}

	FD := myUtil.WebsocketFD(conn)
	fmt.Println("连接的文件描述符", FD)

	client := &myWS.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()

}

func ginWs(pool *myWS.Pool, c *gin.Context) {
	fmt.Println("WedSocket EndPoint Hit")

	conn, err := myWS.Upgrade(c.Writer, c.Request)
	if err != nil {
		fmt.Fprintf(c.Writer, "%+v\n", err)
	} else {
		log.Println("成功建立连接")
	}

	FD := myUtil.WebsocketFD(conn)
	fmt.Println("连接的文件描述符", FD)

	client := &myWS.Client{
		Conn: conn,
		Pool: pool,
	}

	pool.FDs[FD] = client
	pool.Register <- client

	epoller.Add(conn)

	// client.Read()

}

// 添加路由
func setupRoutes(r *gin.Engine) *gin.Engine {

	pool := myWS.NewPool()
	go pool.Start()

	go poll(epoller, pool)
	r.GET("/ws", func(c *gin.Context) {
		ginWs(pool, c)
	})

	// http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
	// 	serveWs(pool, w, r)
	// })

	return r
}

func poll(epoller myEpoll.Epoller, pool *myWS.Pool) {
	for {
		conns, err := epoller.Wait(128)
		if err != nil {
			if err.Error() != "bad file descriptior" {
				log.Println("poll 失败", err)
			}
			continue
		}

		for _, conn := range conns {
			if conn == nil {
				break
			}
			fd := myUtil.WebsocketFD(conn)
			log.Println("在处理消息 连接:", fd)
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				if err := epoller.Remove(conn); err != nil {
					log.Println("停止监听连接失败, 文件描述符: ", fd)
				} else {
					log.Println("停止监听连接成功, 文件描述符: ", fd)
				}

				client := pool.FDs[fd]
				pool.Unregister <- client // 向pool发送注销用户
				conn.Close()              // 关闭websocket连接

			}
			FD := myUtil.WebsocketFD(conn)

			message := myWS.Message{Fd: FD, Type: messageType, Body: string(p)}

			if string(p) == "Happy" {
				pool.Push <- message
			} else {
				pool.Broadcast <- message
			}

			fmt.Printf("Message Received: %+v\n", message)
		}
	}
}

func main() {

	var err error
	epoller, err = myEpoll.NewEpoller()
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	setupRoutes(r)
	// http.ListenAndServe(":8080", nil)
	r.Run(":8080")
}
