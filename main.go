package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const conundrumDuration = 30

var conundrums = []Conundrum{
	{
		Answer:  "GRAPPLING",
		Jumbled: "PIGGAPNLR",
		Hint:    "Struggle without weapons",
	},
	{
		Answer:  "MOMENTARY",
		Jumbled: "TYRMANOME",
		Hint:    "Over in a jiffy",
	},
	{
		Answer:  "HALLMARKS",
		Jumbled: "SLRMKHAAL",
		Hint:    "Certify that purity",
	},
	{
		Answer:  "HANDIWORK",
		Jumbled: "RAIWDHOKN",
		Hint:    "Wowee - you made that?",
	},
	{
		Answer:  "EVERGREEN",
		Jumbled: "VEEEGERRN",
		Hint:    "Leafy boys",
	},
	{
		Answer:  "DIGNITIES",
		Jumbled: "IENIDISTG",
		Hint:    "Self respect",
	},
	{
		Answer:  "ECOSYSTEM",
		Jumbled: "ESSEOTMCY",
		Hint:    "Interconnected system",
	},
	{
		Answer:  "INHIBITED",
		Jumbled: "NTIIBDHIE",
		Hint:    "hold back",
	},
	{
		Answer:  "MUDDINESS",
		Jumbled: "DEIUSDNSM",
		Hint:    "Sloppy wet earthiness",
	},
	{
		Answer:  "SLINGSHOT",
		Jumbled: "TNHGLSISO",
		Hint:    "Forked stick + elastic",
	},
	{
		Answer:  "DISCOUNTS",
		Jumbled: "CDTNOUSIS",
		Hint:    "Reductions",
	},
	{
		Answer:  "ANXIETIES",
		Jumbled: "EIASENXTI",
		Hint:    "feelings of worry",
	},
	{
		Answer:  "OVERDRAFT",
		Jumbled: "FODATRRVE",
		Hint:    "Deficit",
	},
	{
		Answer:  "SPECTACLE",
		Jumbled: "AECLESCPT",
		Hint:    "Striking performance",
	},
	{
		Answer:  "DAYDREAMS",
		Jumbled: "MYRSDADAE",
		Hint:    "Pleasant thoughts",
	},
	{
		Answer:  "SOLICITOR",
		Jumbled: "TLROCOISI",
		Hint:    "Legal professional",
	},
	{
		Answer:  "HARROWING",
		Jumbled: "IOGWRRNHA",
		Hint:    "Disturbing",
	},
	{
		Answer:  "INTERVIEW",
		Jumbled: "NEWVERTII",
		Hint:    "meeting",
	},
	{
		Answer:  "ASHAMEDLY",
		Jumbled: "LADEYMSHA",
		Hint:    "Feeling disgrace",
	},
	{
		Answer:  "CREATURES",
		Jumbled: "TEEUARCRS",
		Hint:    "Animals",
	},
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var ticker *time.Ticker = time.NewTicker(conundrumDuration * time.Second)
var hintTicker *time.Ticker
var scores = make(map[string]int)

type Conundrum struct {
	Answer  string
	Jumbled string
	Hint    string
}

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

type CurrentConundrum struct {
	mu sync.Mutex
	v  Conundrum
}

func (c *CurrentConundrum) Set(conundrum Conundrum) {
	c.mu.Lock()
	c.v = conundrum
	c.mu.Unlock()
}

func (c *CurrentConundrum) Get() Conundrum {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.v
}

func main() {
	var c CurrentConundrum
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", c.handleConnections)

	go c.handleMessages()
	go c.cycleConundrums()
	go c.hintForConundrum()

	go fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("Error starting server: " + err.Error())
	}
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Chat Room!")
}

func convertScoresToJSON(scores map[string]int) string {
	// Encode the map to JSON bytes
	jsonData, err := json.Marshal(scores)
	if err != nil {
		return ""
	}

	// Convert the JSON bytes to a string
	return string(jsonData)
}

func (c *CurrentConundrum) cycleConundrums() {
	for {
		select {
		case <-ticker.C:
			newConundrum := conundrums[rand.Intn(len(conundrums))]
			c.Set(newConundrum)
			broadcast <- Message{"Scores", convertScoresToJSON(scores)}
			broadcast <- Message{"SusieDent", newConundrum.Jumbled}
		}
	}
}

func (c *CurrentConundrum) hintForConundrum() {
	offsetDuration := conundrumDuration / 2 * time.Second
	time.Sleep(offsetDuration)
	hintTicker = time.NewTicker(conundrumDuration * time.Second)
	for {
		select {
		case <-hintTicker.C:
			var currentConundrum = c.Get()
			broadcast <- Message{"SusieDentsAssistantThatGivesHints", currentConundrum.Hint}
		}
	}
}

func (c *CurrentConundrum) handleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func (c *CurrentConundrum) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	defer ticker.Stop()

	clients[conn] = true

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println(err)
			delete(clients, conn)
			return
		}
		if _, ok := scores[msg.Username]; !ok {
			scores[msg.Username] = 0
		}

		// Check against current conundrum
		if strings.ToUpper(msg.Message) == c.Get().Answer {
			msg = Message{msg.Username, "Guessed correctly"}
			scores[msg.Username]++
		}

		broadcast <- msg
	}
}
