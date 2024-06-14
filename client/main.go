package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type Block struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to server. You can start sending messages or retrieve blocks.")

	go readMessages(conn)

	for {
		fmt.Print("Enter command (send/retrieve): ")
		reader := bufio.NewReader(os.Stdin)
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		switch command {
		case "send":
			sendUserMessage(conn)
		case "retrieve":
			retrieveBlock(conn)
		default:
			fmt.Println("Unknown command")
		}
	}
}

func sendUserMessage(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a message: ")
		message, _ := reader.ReadString('\n')
		message = strings.TrimSpace(message)

		err := createAndSendBlock(conn, message)
		if err != nil {
			fmt.Println("Error sending message:", err)
		} else {
			break
		}
	}
}

func createAndSendBlock(conn net.Conn, message string) error {
	hash := sha256.Sum256([]byte(message))
	hashString := hex.EncodeToString(hash[:])

	block := Block{
		Hash:    hashString,
		Message: message,
	}
	blockData, err := json.Marshal(block)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(string(blockData) + "\n"))
	if err != nil {
		return err
	}
	return nil
}

func retrieveBlock(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter block index: ")
	index, _ := reader.ReadString('\n')
	index = strings.TrimSpace(index)

	_, err := conn.Write([]byte("retrieve:" + index + "\n"))
	if err != nil {
		fmt.Println("Error requesting block:", err)
		return
	}

	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading block from server:", err)
		return
	}
	fmt.Println("Block from server:", message)
}

func readMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading message from server:", err)
			return
		}
		fmt.Println("Message from server:", message)
	}
}
