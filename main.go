package main

import (
	"log"
	"net/http"
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

type Message struct {
	Join
	Content string `json:"content"`
}

type Join struct {
	From string `json:"from"`
	To   string `json:"to"`
}

var roomChannels = make(map[string]chan Message)
var roomMutex = &sync.Mutex{}

func getRoomName(user1, user2 string) string {
	// Sort names to ensure uniqueness
	if user1 < user2 {
		return user1 + ":" + user2
	}
	return user2 + ":" + user1
}

func handleRoomMessages(roomName string, ch chan Message, server *socketio.Server) {
	for msg := range ch {
		server.BroadcastToRoom("/", roomName, "chat", msg)
	}
}

func main() {
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(so socketio.Conn) error {
		so.SetContext("")
		log.Printf("Connected: %s", so.ID())
		return nil
	})

	server.OnEvent("/", "join", func(so socketio.Conn, join Join) {
		so.Join(getRoomName(join.From, join.To))
	})

	server.OnEvent("/", "chat", func(so socketio.Conn, msg Message) {
		roomName := getRoomName(msg.From, msg.To)
		roomMutex.Lock()
		roomCh, ok := roomChannels[roomName]
		if !ok {
			roomCh = make(chan Message)
			roomChannels[roomName] = roomCh
			go handleRoomMessages(roomName, roomCh, server)
		}
		roomMutex.Unlock()
		roomCh <- msg
	})

	server.OnDisconnect("/", func(so socketio.Conn, reason string) {
		log.Printf("Disconnected: %s", so.ID())
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	log.Println("Serving at :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
