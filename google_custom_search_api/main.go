package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// SearchRequest は検索リクエストを表す構造体です
type SearchRequest struct {
	Query string `json:"query" binding:"required"`
}

// SearchResult は検索結果を表す構造体です
type SearchResult struct {
	Items []SearchItem `json:"items"`
}

// SearchItem は検索結果の各アイテムを表す構造体です
type SearchItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

func main() {
	r := gin.Default()

	// エンドポイントの設定
	r.POST("/search", handleSearch)

	// ポートの設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// サーバーの起動
	r.Run(":" + port)
}

// handleSearch は検索リクエストを処理するハンドラーです
func handleSearch(c *gin.Context) {
	// request body から検索クエリを取得
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"リクエストが正しくありません": err.Error()})
		return
	}

	// Google Custom Search API を使って検索を実行
	results, err := performGoogleSearch(c, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ハンドラー内でエラーが発生しました": err.Error()})
		return
	}

	// レスポンスを返す
	c.JSON(http.StatusOK, results)
}

// performGoogleSearch は Google Custom Search API を使って検索を実行します
func performGoogleSearch(ctx context.Context, query string) (*SearchResult, error) {
	// env ファイルの読み込み
	err := godotenv.Load()
	if err != nil {
		fmt.Println(".envファイルの読み込みに失敗しました")
	}

	// Google Custom Search API のエンドポイントのsetup
	baseURL := "https://www.googleapis.com/customsearch/v1"
	apiKey := os.Getenv("GOOGLE_API_KEY")
	searchEngineID := os.Getenv("SEARCH_ENGINE_ID")
	if apiKey == "" || searchEngineID == "" {
		return nil, fmt.Errorf("APIキーまたは検索エンジンIDが設定されていません")
	}

	// エンドポイントを作成
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("エンドポイントの作成に失敗しました: %v", err)
	}

	// クエリパラメータを設定
	q := req.URL.Query()
	q.Set("key", apiKey)
	q.Set("cx", searchEngineID)
	q.Set("q", query) // 検索クエリ

	// URL にクエリパラメータをセット
	req.URL.RawQuery = q.Encode()

	// クライアントの作成
	client := &http.Client{}

	// Google Custom Search API にリクエストを送信
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("”リクエストの送信に失敗しました: %v", err)
	}
	defer resp.Body.Close()

	// Google Custom Search API からのレスポンスをチェック
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API からのレスポンスがエラーです: %s", body)
	}

	// レスポンスのデコード
	var searchResponse struct {
		Items []SearchItem `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("デコードに失敗しました: %v", err)
	}

	// 検索結果を返す
	return &SearchResult{
		Items: searchResponse.Items,
	}, nil
}
