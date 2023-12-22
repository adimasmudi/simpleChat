package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	gubrak "github.com/novalagung/gubrak/v2"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type M map[string]interface{}

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From           string
	Type           string
	Message        string
	TotalAttendees string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var totalAttendee = 0

var connections = make([]*WebSocketConnection, 0)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile("index.html")

		if err != nil {
			http.Error(w, "Could not open requested file", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%s", content)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		currentGorillaConn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)

		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		}

		username := r.URL.Query().Get("username")
		currentConn := WebSocketConnection{Conn: currentGorillaConn, Username: username}
		connections = append(connections, &currentConn)

		totalAttendee = len(connections)

		go handleIO(&currentConn, connections)

	})

	fmt.Println("Server starting at PORT 5000")

	http.ListenAndServe(":5000", nil)
}

func handleIO(currentConn *WebSocketConnection, connections []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	broadcastMessage(currentConn, MESSAGE_NEW_USER, "", totalAttendee)

	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)

		fmt.Println("payload", payload)

		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				broadcastMessage(currentConn, MESSAGE_LEAVE, "", totalAttendee)
				ejectConnection(currentConn)
				return
			}

			log.Println("ERROR", err.Error())
			continue
		}

		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message, totalAttendee)
	}
}

func ejectConnection(currentConn *WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *WebSocketConnection) bool {
		return each == currentConn
	}).Result()

	connections = filtered.([]*WebSocketConnection)

}

func broadcastMessage(currentConn *WebSocketConnection, kind, message string, totalAttendee int) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}

		totalAttendees := strconv.Itoa(totalAttendee)

		eachConn.WriteJSON(SocketResponse{
			From:           currentConn.Username,
			Type:           kind,
			Message:        message,
			TotalAttendees: totalAttendees,
		})
	}

}
