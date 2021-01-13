package main

import (
	"fmt"
	"log"
	"net/http"

	myUtil "Realtime/pkg/utils"
	myWS "Realtime/pkg/websocket"

	"github.com/gin-gonic/gin"
)

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
	client.Read()

}

// 添加路由
func setupRoutes(r *gin.Engine) *gin.Engine {

	pool := myWS.NewPool()
	go pool.Start()
	r.GET("/ws", func(c *gin.Context) {
		ginWs(pool, c)
	})

	// http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
	// 	serveWs(pool, w, r)
	// })

	return r
}

func main() {
	r := gin.Default()
	setupRoutes(r)
	// http.ListenAndServe(":8080", nil)
	r.Run(":8080")
}
