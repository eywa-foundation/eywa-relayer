package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/zishang520/engine.io/types"
	"github.com/zishang520/socket.io/socket"
)

type Message struct {
	Join    `mapstructure:",squash" json:",inline"`
	Content string `json:"content"`
}

type Join struct {
	From string `json:"from"`
	To   string `json:"to"`
}

var roomChannels = make(map[socket.Room]chan Message)
var roomMutex = &sync.Mutex{}

func getRoomName(user1, user2 string) socket.Room {
	// Sort names to ensure uniqueness
	if user1 < user2 {
		return socket.Room(user1 + ":" + user2)
	}
	return socket.Room(user2 + ":" + user1)
}

func handleRoomMessages(roomName socket.Room, ch chan Message, server *socket.Server) {
	for msg := range ch {
		server.Of(roomName, nil).Emit("chat", msg)
	}
}

func main() {
	c := socket.DefaultServerOptions()
	c.SetCors(&types.Cors{
		Origin:      true,
		Credentials: true,
	})
	io := socket.NewServer(nil, c)
	io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		log.Printf("Connected: %s", client.Id())

		client.On("join", func(datas ...any) {
			join := Join{}
			mapstructure.Decode(datas[0], &join)
			room := getRoomName(join.From, join.To)
			client.Join(room)
		})
		client.On("chat", func(datas ...any) {
			msg := Message{}
			mapstructure.Decode(datas[0], &msg)
			roomName := getRoomName(msg.From, msg.To)
			roomMutex.Lock()
			roomCh, ok := roomChannels[roomName]
			if !ok {
				roomCh = make(chan Message)
				roomChannels[roomName] = roomCh
				go handleRoomMessages(roomName, roomCh, io)
			}
			roomMutex.Unlock()
			roomCh <- msg
		})
		client.On("disconnect", func(...any) {
			log.Printf("Disconnected: %s", client.Id())
		})
	})

	defer io.Close(nil)

	http.Handle("/socket.io/", io.ServeHandler(nil))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Println("Serving at :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
