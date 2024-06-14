package main

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Block struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

var (
	peers = make([]net.Conn, 0)
	mutex sync.Mutex
	db    *sql.DB
)

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./blockchain.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	createTable()

	go startServer()

	go func() {
		for {
			time.Sleep(10 * time.Second) // Adjust the interval as needed
			captureAndPrintMessages()
		}
	}()

	sendUserMessage()
}

func createTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS blocks (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"hash" TEXT,
		"message" TEXT
	);`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		fmt.Println("Error creating table:", err)
	}
}

func startServer() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server started. Waiting for connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		mutex.Lock()
		peers = append(peers, conn)
		mutex.Unlock()

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}
		fmt.Println("Message received:", message)
		err = createAndStoreBlock(message)
		if err != nil {
			fmt.Println("Error storing block:", err)
		}
		err = sendMessageToAllPeers(message)
		if err != nil {
			fmt.Println("Error forwarding message to peers:", err)
		}
	}
}

func sendUserMessage() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter a message: ")
		message, _ := reader.ReadString('\n')

		err := createAndStoreBlock(message)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}
}

func createAndStoreBlock(message string) error {
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
	err = storeBlockInDB(block)
	if err != nil {
		return err
	}
	return sendMessageToAllPeers(string(blockData))
}

func storeBlockInDB(block Block) error {
	insertSQL := `INSERT INTO blocks (hash, message) VALUES (?, ?)`
	_, err := db.Exec(insertSQL, block.Hash, block.Message)
	if err != nil {
		return err
	}
	return nil
}

func sendMessageToAllPeers(message string) error {
	mutex.Lock()
	defer mutex.Unlock()

	for _, peer := range peers {
		_, err := peer.Write([]byte(message + "\n"))
		if err != nil {
			fmt.Println("Error sending message to peer:", err)
			continue
		}
	}
	return nil
}

func captureAndPrintMessages() {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Println("Captured Messages from Blockchain:")
	rows, err := db.Query("SELECT hash, message FROM blocks")
	if err != nil {
		fmt.Println("Error querying database:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var hash, message string
		err := rows.Scan(&hash, &message)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			continue
		}
		fmt.Printf("Hash: %s, Message: %s\n", hash, message)
	}
}
