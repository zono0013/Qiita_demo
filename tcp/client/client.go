package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// サーバーに接続
	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	// 受信を別ゴルーチンで処理
	go func() {
		for {
			message, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Print(message)
		}
	}()

	// メッセージ送信ループ
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter message: ")
	for {
		text, _ := reader.ReadString('\n')
		conn.Write([]byte(text))
	}
}
