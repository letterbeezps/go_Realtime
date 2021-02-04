package epoll

import (
	"log"
	"sync"
	"syscall"

	myUtil "Realtime/pkg/utils"

	"github.com/gorilla/websocket"
)

type epoll struct {
	fd          int
	ts          syscall.Timespec
	changes     []syscall.Kevent_t
	connections map[int]*websocket.Conn
	lock        *sync.RWMutex
	connbuf     []*websocket.Conn
	events      []syscall.Kevent_t
}

func NewEpoller() (Epoller, error) {
	p, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	_, err = syscall.Kevent(p, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}

	return &epoll{
		fd:          p,
		ts:          syscall.NsecToTimespec(1e9),
		lock:        &sync.RWMutex{},
		connbuf:     make([]*websocket.Conn, 128, 128),
		events:      make([]syscall.Kevent_t, 128, 128),
		connections: make(map[int]*websocket.Conn),
	}, nil
}

func (e *epoll) Add(conn *websocket.Conn) error {
	fd := myUtil.WebsocketFD(conn)

	e.lock.Lock()
	defer e.lock.Unlock()
	e.changes = append(e.changes, syscall.Kevent_t{
		Ident: uint64(fd), Flags: syscall.EV_ADD | syscall.EV_EOF, Filter: syscall.EVFILT_READ,
	})
	log.Println("连接的数目: ", len(e.changes))
	e.connections[fd] = conn
	return nil
}

func (e *epoll) Remove(conn *websocket.Conn) error {
	fd := myUtil.WebsocketFD(conn)

	e.lock.Lock()
	e.lock.Unlock()

	if len(e.changes) <= 1 {
		e.changes = nil
	} else {
		changes := make([]syscall.Kevent_t, 0, len(e.changes)-1)
		ident := uint64(fd)
		for _, ke := range e.changes {
			if ke.Ident != ident {
				changes = append(changes)
			}
		}
		e.changes = changes
	}
	delete(e.connections, fd)
	return nil
}

func (e *epoll) Wait(count int) ([]*websocket.Conn, error) {
	events := make([]syscall.Kevent_t, count, count)
	e.lock.RLock()
	changes := e.changes
	e.lock.RUnlock()

retry:
	log.Println("开始监听")
	n, err := syscall.Kevent(e.fd, changes, events, &e.ts)
	if err != nil {
		if err == syscall.EINTR {
			goto retry
		}
		return nil, err
	}

	var connections = make([]*websocket.Conn, 0, n)
	e.lock.RLock()
	for i := 0; i < n; i++ {
		conn := e.connections[int(events[i].Ident)]
		if (events[i].Flags & syscall.EV_EOF) == syscall.EV_EOF {
			conn.Close()
		}
		log.Println("epoll监听到连接有消息 文件描述符: ", events[i].Ident)
		connections = append(connections, conn)
	}
	e.lock.RUnlock()

	return connections, nil
}
