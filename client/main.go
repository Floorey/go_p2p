package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Block struct {
	Hash    string `json:"hash"`
	Message string `json:"message`
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	go readMessages(conn)

	fmt.Println("Connected to server. You can start sending messages.")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a message: ")
		message, _ := reader.ReadString('\n')

		err := createAndSendBlock(conn, message)
		if err != nil {
			fmt.Println("Error sending message:", err)
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
