package main

import (
	"os"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"net/url"
	"github.com/joho/godotenv"
)

const (
	WebSocketUrl = "ws://s.rabbitmq.cedric.5y5.one:16666"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


// RabbitMQ consumer
// RabbitMQ consumer
func consumeFromRabbitMQ(queueName string, messages chan<- string) {
        adminUserName := getRabbitMqAdmin()
	adminUserPassword := getRabbitMqPassword()
	rabbitMqUrl := getRabbitMqUrl()
        var RabbitMQUrl = fmt.Sprintf("amqp://%s:%s@%s:5672", adminUserName, url.QueryEscape(adminUserPassword), rabbitMqUrl)
	conn, err := amqp.Dial(RabbitMQUrl)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclarePassive(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare or inspect a queue: %v", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	for msg := range msgs {
		messages <- string(msg.Body)
	}
}

// WebSocket publisher
func publishToWebSocket(messages <-chan string, clients map[*websocket.Conn]bool) {
	for message := range messages {
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Failed to write message to client:", err)
				delete(clients, client)
				client.Close()
			}
		}
	}
}

// WebSocket handler function
func handleWebSocket(clients map[*websocket.Conn]bool, w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection to WebSocket:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	// Read messages from WebSocket
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read message from WebSocket:", err)
			delete(clients, conn)
			break
		}
	}
}

func main() {
	messages := make(chan string)
	clients := make(map[*websocket.Conn]bool)

	// Set up the WebSocket route
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(clients, w, r)
	})

	// Start the WebSocket server
	go func() {
		err := http.ListenAndServe(":16666", nil)
		if err != nil {
			log.Fatal("Failed to start WebSocket server:", err)
		}
	}()

	go consumeFromRabbitMQ("00000000001", messages)

	publishToWebSocket(messages, clients)
}

func getRabbitMqAdmin() string {
	err := godotenv.Load()
	if err != nil {
		return ""
	}
	return os.Getenv("RABBITMQ_ADMIN")
}

func getRabbitMqUrl() string {
	err := godotenv.Load()
	if err != nil {
		return ""
	}

	return os.Getenv("RABBITMQ_HOST")
}

func getRabbitMqPassword() string {
	err := godotenv.Load()
	if err != nil {
		return ""
	}

	return  os.Getenv("RABBITMQ_PASSWORD")
}
