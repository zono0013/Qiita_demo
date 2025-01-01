package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gocv.io/x/gocv"
	"hash/crc32"
	"log"
	"net"
	"sync"
)

type PacketHeader struct {
	ImageID      uint32
	SequenceNum  uint16
	TotalPackets uint16
	PayloadSize  uint16
	Checksum     uint32
}

type ImageAssembler struct {
	currentImage    map[uint16][]byte
	receivedPackets map[uint16]struct{}
	imageID         uint32
	totalPackets    uint16
	mu              sync.Mutex
}

func NewImageAssembler() *ImageAssembler {
	return &ImageAssembler{
		currentImage:    make(map[uint16][]byte),
		receivedPackets: make(map[uint16]struct{}),
	}
}

func (ia *ImageAssembler) reset() {
	ia.currentImage = make(map[uint16][]byte)
	ia.receivedPackets = make(map[uint16]struct{})
}

func main() {
	// サーバーのアドレス解決
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	// ローカルアドレス解決
	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	// UDP接続
	conn, err := net.DialUDP("udp", localAddr, serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Connected to server")

	// サーバーに登録
	_, err = conn.Write([]byte("register"))
	if err != nil {
		log.Fatal("Registration failed:", err)
	}

	// 画像表示用のウィンドウを作成
	window := gocv.NewWindow("UDP Stream")
	defer window.Close()

	assembler := NewImageAssembler()
	receiveAndDisplay(conn, assembler, window)
}

func (ia *ImageAssembler) addPacket(header PacketHeader, payload []byte) ([]byte, bool) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	// 新しい画像が始まった場合はリセット
	if header.ImageID != ia.imageID {
		log.Printf("New image started: ID=%d, Total packets=%d", header.ImageID, header.TotalPackets)
		ia.reset()
		ia.imageID = header.ImageID
		ia.totalPackets = header.TotalPackets
	}

	// チェックサムの確認
	if crc32.ChecksumIEEE(payload) != header.Checksum {
		log.Printf("Checksum mismatch for packet %d of image %d", header.SequenceNum, header.ImageID)
		return nil, false
	}

	log.Printf("Sequence number: %d, Payload First 16 bytes: % x", header.SequenceNum, payload[:16])

	// パケットの保存
	copiedPayload := make([]byte, len(payload))
	copy(copiedPayload, payload)

	// `ia.currentImage` にコピーを格納
	ia.currentImage[header.SequenceNum] = copiedPayload
	if header.SequenceNum >= 1 {
		log.Printf("First 16 bytes of encoded image: % x", ia.currentImage[1][:16])
	}
	ia.receivedPackets[header.SequenceNum] = struct{}{}

	log.Printf("Received packet %d/%d for image %d", header.SequenceNum, header.TotalPackets, header.ImageID)

	// 全パケットが揃ったかチェック
	if len(ia.receivedPackets) == int(ia.totalPackets) {
		log.Printf("All packets received for image %d, assembling...", header.ImageID)
		var completeImage []byte

		// パケットを順番に結合
		for i := uint16(0); i < ia.totalPackets; i++ {
			if data, ok := ia.currentImage[i]; ok {
				completeImage = append(completeImage, data...)
			} else {
				log.Printf("Missing packet %d in sequence for image %d", i, header.ImageID)
				return nil, false
			}
		}

		log.Printf("Image %d assembled, total size: %d bytes", header.ImageID, len(completeImage))
		ia.reset()
		return completeImage, true
	}

	return nil, false
}

// クライアント側の receiveAndDisplay 関数の修正
func receiveAndDisplay(conn *net.UDPConn, assembler *ImageAssembler, window *gocv.Window) {
	buffer := make([]byte, 65507)
	headerSize := binary.Size(PacketHeader{})

	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		if n <= headerSize {
			log.Printf("Received packet too small: %d bytes", n)
			continue
		}

		// ヘッダーのデシリアライズ
		var header PacketHeader
		headerBuf := bytes.NewReader(buffer[:headerSize])
		if err := binary.Read(headerBuf, binary.BigEndian, &header); err != nil {
			log.Printf("Header decode error: %v", err)
			continue
		}

		// ペイロードの抽出
		payload := buffer[headerSize:n]

		// 画像の組み立てを試行
		if completeImage, ok := assembler.addPacket(header, payload); ok {
			// 最初の数バイトをデバッグ出力
			if len(completeImage) > 16 {
				log.Printf("First 16 bytes of complete image: % x", completeImage[:16])
			}

			// JPEGファイルヘッダーの確認
			if len(completeImage) < 2 || completeImage[0] != 0xFF || completeImage[1] != 0xD8 {
				log.Printf("Invalid JPEG header")
				continue
			}

			// 画像のデコード
			mat, err := gocv.IMDecode(completeImage, gocv.IMReadColor)
			if err != nil {
				log.Printf("Image decode error: %v", err)
				continue
			}

			if mat.Empty() {
				log.Printf("Decoded image is empty")
				mat.Close()
				continue
			}

			size := mat.Size()
			log.Printf("Successfully decoded image: %dx%d", size[0], size[1])

			window.IMShow(mat)
			if window.WaitKey(1) >= 0 {
				mat.Close()
				return
			}
			mat.Close()
		}
	}
}
