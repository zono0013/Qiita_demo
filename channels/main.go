package main

import (
	"fmt"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- int) {
	for job := range jobs {
		fmt.Printf("Worker %d processing job %d\n", id, job)
		time.Sleep(time.Second)
		results <- job * 2
	}
}

func main() {
	jobs := make(chan int, 10)
	results := make(chan int, 10)

	// 3つのワーカーゴルーチンを起動
	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results)
	}

	// ジョブを送信
	for j := 1; j <= 5; j++ {
		jobs <- j
	}
	close(jobs)

	// 結果を収集
	for a := 1; a <= 5; a++ {
		fmt.Println(<-results)
	}
}
