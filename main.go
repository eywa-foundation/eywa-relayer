package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/cosmos/btcutil/bech32"
	"github.com/eywa-foundation/eywaclient"
	"github.com/joho/godotenv"
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
		return socket.Room(user1 + "_" + user2)
	}
	return socket.Room(user2 + "_" + user1)
}

func isCosmosAddress(address string) bool {
	if !strings.HasPrefix(address, "cosmos") {
		return false
	}
	_, _, err := bech32.Decode(address, 128)
	return err == nil
}

func handleRoomMessages(roomName socket.Room, ch chan Message, server *socket.Server) {
	for msg := range ch {
		server.Of(roomName, nil).Emit("chat", msg)
	}
}

func getEnv() (string, string, string) {
	godotenv.Load()
	accountName := os.Getenv("ACCOUNT_NAME")
	nodeAddress := os.Getenv("NODE_ADDRESS")
	mnemonic := os.Getenv("MNEMONIC")
	if accountName == "" || nodeAddress == "" || mnemonic == "" {
		log.Fatal("ACCOUNT_NAME or NODE_ADDRESS is not set")
	}
	log.Println("ACCOUNT_NAME:", accountName)
	log.Println("NODE_ADDRESS:", nodeAddress)
	log.Println("MNEMONIC:", mnemonic)
	return accountName, mnemonic, nodeAddress
}

func main() {
	accountName, mnemonic, nodeAddress := getEnv()

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
			// Only allow join if both addresses are valid
			if !isCosmosAddress(join.From) || !isCosmosAddress(join.To) {
				return
			}
			room := getRoomName(join.From, join.To)
			client.Join(room)
		})

		client.On("chat", func(datas ...any) {
			msg := Message{}
			mapstructure.Decode(datas[0], &msg)
			if !isCosmosAddress(msg.From) || !isCosmosAddress(msg.To) {
				return
			}

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

			// Broadcast Tx in goroutines
			// WARNING: This is not safe, but it's just a demo
			// and must set mnemonic properly that is knwon to ignite chain
			go eywaclient.CreateChatTx(
				nodeAddress,
				accountName,
				mnemonic,
				string(getRoomName(msg.From, msg.To)),
				// NOTE: msg.From must be known address in chain
				msg.From,
				msg.To,
				msg.Content)

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
