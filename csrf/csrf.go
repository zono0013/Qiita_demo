package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

var store *sessions.CookieStore

// CSRFトークンを生成する関数
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// CSRFトークンを検証するミドルウェア
func csrfMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "csrf-session")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" {
			// ヘッダーからCSRFトークンを取得
			formToken := r.Header.Get("X-CSRF-Token")

			// セッションに保存されているトークンを取得
			sessionToken, ok := session.Values["csrf_token"].(string)

			if !ok || formToken == "" || formToken != sessionToken {
				http.Error(w, "Invalid CSRF Token", http.StatusForbidden)
				return
			}
		}

		next(w, r)
	}
}

// セッション生成とCSRFトークンの設定
func createSession(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "csrf-session")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	token := generateToken()
	session.Values["csrf_token"] = token
	store.Options = &sessions.Options{
		HttpOnly: true, // JavaScriptからアクセスできないようにする
		Secure:   true, // HTTPS通信のみに送信されるようにする
		MaxAge:   3600, // クッキーの有効期限（例：1時間）
	}
	if err := session.Save(r, w); err != nil {
		fmt.Fprintf(w, "error: Failed to save session: %s", err)
		return
	}

	// トークンを返す（ここでは単に通知として返す）
	fmt.Fprintf(w, "Session created. CSRF Token: %s", token)
}

// JSONリクエストを処理するハンドラ
func submitForm(w http.ResponseWriter, r *http.Request) {
	var requestData map[string]string
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	message := requestData["message"]
	fmt.Fprintf(w, "Received message: %s", message)
}

func main() {
	// .env ファイルをロード
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	store = sessions.NewCookieStore([]byte(os.Getenv("SECRET_KEY"))) // セッションストアの作成
	store.Options = &sessions.Options{
		HttpOnly: true,
		Secure:   true, // HTTPS通信のみに送信される
		MaxAge:   600,  // クッキーの有効期限(10分)
		SameSite: http.SameSiteStrictMode,
	}

	http.HandleFunc("/create-session", createSession)
	http.HandleFunc("/submit", csrfMiddleware(submitForm))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
