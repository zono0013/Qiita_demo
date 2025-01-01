package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

// プレイヤー情報を格納する構造体
type Player struct {
	ID   string
	Conn net.Conn
}

var (
	players  = make(map[string]Player) // プレイヤーの管理
	mutex    = sync.Mutex{}            // 排他的制御のためのMutex
	playerID = 0                       // プレイヤーIDを生成するカウンタ
)

// クライアントの接続を処理する
func handleClient(conn net.Conn) {
	defer conn.Close()

	// プレイヤーIDを割り当てる
	mutex.Lock()
	playerID++
	id := fmt.Sprintf("Player%d", playerID)
	players[id] = Player{ID: id, Conn: conn}
	mutex.Unlock()

	fmt.Printf("%s connected\n", id)
	broadcastMessage(fmt.Sprintf("%s has joined the game!\n", id), id)

	reader := bufio.NewReader(conn)
	for {
		// クライアントからのメッセージを読み取る
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected\n", id)
			removePlayer(id)
			broadcastMessage(fmt.Sprintf("%s has left the game.\n", id), "")
			return
		}

		// メッセージを処理して他のプレイヤーに通知
		message = strings.TrimSpace(message)
		fmt.Printf("[%s]: %s\n", id, message)
		broadcastMessage(fmt.Sprintf("[%s]: %s\n", id, message), id)
	}
}

// 他のプレイヤーにメッセージを送信する
func broadcastMessage(message string, senderID string) {
	mutex.Lock()
	defer mutex.Unlock()
	for id, player := range players {
		if id != senderID { // 送信者には送り返さない
			player.Conn.Write([]byte(message))
		}
	}
}

// プレイヤーを削除する
func removePlayer(id string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(players, id)
}

func main() {
	// サーバーの起動
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal("Server start error:", err)
	}
	defer listener.Close()
	fmt.Println("Game server listening on :8000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleClient(conn)
	}
}
