package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"gocv.io/x/gocv"
)

type Client struct {
	addr *net.UDPAddr
}

type PacketHeader struct {
	ImageID      uint32
	SequenceNum  uint16
	TotalPackets uint16
	PayloadSize  uint16
	Checksum     uint32
}

var (
	clients        = make(map[string]*Client)
	clientsMu      sync.Mutex
	currentImageID uint32
)

func main() {
	fmt.Println("Starting server...")

	// WebカメラをOpen
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		log.Fatal("Error opening webcam:", err)
	}
	defer webcam.Close()

	// UDPサーバーの設定
	addr, err := net.ResolveUDPAddr("udp", ":8000")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Server started on :8000")

	// クライアント登録の処理
	go handleClientRegistration(conn)

	// 画像のストリーミング処理
	streamImages(webcam, conn)
}

func handleClientRegistration(conn *net.UDPConn) {
	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		if string(buffer[:n]) == "register" {
			clientsMu.Lock()
			clientKey := remoteAddr.String()
			clients[clientKey] = &Client{addr: remoteAddr}
			clientsMu.Unlock()

			log.Printf("New client registered: %s", clientKey)
		}
	}
}

func streamImages(webcam *gocv.VideoCapture, conn *net.UDPConn) {
	img := gocv.NewMat()
	defer img.Close()

	for {
		if ok := webcam.Read(&img); !ok {
			log.Println("Error capturing frame")
			continue
		}

		// 画像が空でないことを確認
		if img.Empty() {
			log.Println("Captured frame is empty")
			continue
		}

		// デバッグ情報の出力
		size := img.Size()
		log.Printf("Captured frame size: %dx%d", size[0], size[1])

		// JPEG形式でエンコード
		buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
		if err != nil {
			log.Println("Encoding error:", err)
			continue
		}

		imageData := buf.GetBytes()

		log.Printf("Encoded image size: %d bytes", len(imageData))

		// 最初の数バイトをデバッグ出力
		if len(imageData) > 16 {
			log.Printf("First 16 bytes of encoded image: % x", imageData[:16])
		}

		// 各クライアントに送信
		clientsMu.Lock()
		for _, client := range clients {
			sendImage(conn, imageData, client)
		}
		clientsMu.Unlock()

		time.Sleep(33 * time.Millisecond)
	}
}

func sendImage(conn *net.UDPConn, imageData []byte, client *Client) {
	packetSize := 1024
	imageID := atomic.AddUint32(&currentImageID, 1)
	totalPackets := uint16((len(imageData) + packetSize - 1) / packetSize)
	log.Printf("imageID: %d", imageID)
	log.Printf("totalPackets: %d", totalPackets)
	log.Printf("packetSize: %d", len(imageData))

	for seqNum := uint16(0); seqNum < totalPackets; seqNum++ {
		start := int(seqNum) * packetSize
		end := start + packetSize
		if end > len(imageData) {
			end = len(imageData)
		}

		if start == 0 {
			log.Printf("First 16 bytes of encoded image: % x", imageData[:16])
		}

		payload := imageData[start:end]
		header := PacketHeader{
			ImageID:      imageID,
			SequenceNum:  seqNum,
			TotalPackets: totalPackets,
			PayloadSize:  uint16(len(payload)),
			Checksum:     crc32.ChecksumIEEE(payload),
		}

		// ヘッダーとペイロードをシリアライズ
		var packet bytes.Buffer
		binary.Write(&packet, binary.BigEndian, header)
		packet.Write(payload)

		_, err := conn.WriteToUDP(packet.Bytes(), client.addr)
		if err != nil {
			log.Printf("Error sending to client %v: %v", client.addr, err)
		}
	}
}
