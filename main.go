package main

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

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

	select {} // Keep the server running
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
		message = strings.TrimSpace(message)
		if strings.HasPrefix(message, "retrieve:") {
			indexStr := strings.TrimPrefix(message, "retrieve:")
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				fmt.Println("Invalid index")
				return
			}
			retrieveAndSendBlock(conn, index)
		} else {
			fmt.Println("Message received:", message)
			err = createAndStoreBlock(message)
			if err != nil {
				fmt.Println("Error storing block:", err)
			}
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

func retrieveAndSendBlock(conn net.Conn, index int) {
	query := `SELECT hash, message FROM blocks WHERE id = ?`
	row := db.QueryRow(query, index)

	var hash, message string
	err := row.Scan(&hash, &message)
	if err != nil {
		if err == sql.ErrNoRows {
			conn.Write([]byte("Block not found\n"))
		} else {
			conn.Write([]byte("Error retrieving block\n"))
		}
		return
	}

	block := Block{
		Hash:    hash,
		Message: message,
	}
	blockData, err := json.Marshal(block)
	if err != nil {
		conn.Write([]byte("Error marshalling block data\n"))
		return
	}

	conn.Write([]byte(string(blockData) + "\n"))
}
