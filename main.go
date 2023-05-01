package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/gousb"
	"log"
	"net/http"
	"os"
	"time"
)

type EscPos struct {
	Ep *gousb.OutEndpoint

	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	// state toggles GS[char]
	reverse, smooth uint8
}
type Token struct {
	Token string
}

func main() {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(0x0416, 0x5011)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	defer dev.Close()

	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", dev, err)
	}
	defer done()

	ep, err := intf.OutEndpoint(1)
	if err != nil {
		log.Fatalf("%s.OutEndpoint(1): %v", intf, err)
	}
	e := &EscPos{
		Ep: ep,
	}
	for {
		e.Init()
		token, err := getToken()
		if err != nil {
			log.Fatal(err)
		}
		messages := getData(token)
		for _, message := range messages {
			log.Println(message)
			e.Write(message)
			e.Formfeed()
			e.Cut()
		}
		err = deleteMessages(token)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(10 * time.Second)
		e.End()
	}
}
func (e *EscPos) Init() {
	e.Reset()
	e.Write("\x1B@")
}
func (e *EscPos) Reset() {
	e.width = 1
	e.height = 1

	e.underline = 0
	e.emphasize = 0
	e.upsidedown = 0
	e.rotate = 1

	e.reverse = 0
	e.smooth = 0
}
func (e *EscPos) Write(data string) (n int, err error) {
	return e.WriteRaw([]byte(data))
}
func (e *EscPos) WriteRaw(data []byte) (n int, err error) {
	if len(data) > 0 {
		log.Printf("Writing %d bytes\n", len(data))
		e.Ep.Write(data)
	} else {
		log.Printf("Wrote NO bytes\n")
	}
	return 0, nil
}

func (e *EscPos) Cut() {
	e.Write("\x1DVA0")
}

func (e *EscPos) End() {
	e.Write("\xFA")
}
func (e *EscPos) FormfeedN(n int) {
	e.Write(fmt.Sprintf("\x1Bd%c", n))
}

// send formfeed
func (e *EscPos) Formfeed() {
	e.FormfeedN(2)
}
func getData(token string) []string {
	url := os.Getenv("TODO_URL")
	if url == "" {
		log.Fatal("need to set TODO_URL")
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/messages", url), nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "todo-printer/1.0")
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()
	var messages []string
	err = json.NewDecoder(resp.Body).Decode(&messages)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Println(messages)
	return messages
}

func getToken() (string, error) {
	url := os.Getenv("TODO_URL")
	if url == "" {
		log.Fatal("need to set TODO_URL")
	}
	username := os.Getenv("TODO_USERNAME")
	if url == "" {
		log.Fatal("need to set TODO_USERNAME")
	}
	password := os.Getenv("TODO_PASSWORD")
	if url == "" {
		log.Fatal("need to set TODO_PASSWORD")
	}
	resp, err := http.Post(fmt.Sprintf("%s/login", url), "application/json", bytes.NewBuffer([]byte(`{"username":"`+username+`", "password": "`+password+`"}`)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var token Token
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", err
	}

	return token.Token, nil
}

func deleteMessages(token string) error {
	url := os.Getenv("TODO_URL")
	if url == "" {
		log.Fatal("need to set TODO_URL")
	}

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/messages", url), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "todo-printer/1.0")
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	log.Println(resp.StatusCode)

	return nil
}
