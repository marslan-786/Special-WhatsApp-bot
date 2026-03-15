package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mattn/go-sqlite3" // SQLite ڈرائیور
	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// ریلوے کے ویب ساکٹ کو زندہ رکھنے کے لیے اپ گریڈر
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var waClient *whatsmeow.Client

func initDatabase() *sqlstore.Container {
	// ریلوے والیوم کے لیے /data فولڈر (اگر لوکل چلا رہے ہیں تو خود فولڈر بنا لے گا)
	dbPath := "./data"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		os.Mkdir(dbPath, 0755)
	}

	dbLog := waLog.Stdout("Database", "WARN", true)
	// SQLite کنکشن
	container, err := sqlstore.New("sqlite3", "file:./data/whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return container
}

func pairHandler(w http.ResponseWriter, r *http.Request) {
	// CORS ہیڈرز
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Only POST allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Number string `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// اگر پہلے سے لاگ ان ہے
	if waClient.Store.ID != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Already logged in!"})
		return
	}

	// کلائنٹ کنیکٹ کریں
	if !waClient.IsConnected() {
		err := waClient.Connect()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to connect to WhatsApp servers"})
			return
		}
		// کنیکٹ ہونے کے بعد تھوڑا سا ڈیلے (تاکہ سرور ریڈی ہو جائے)
		time.Sleep(2 * time.Second)
	}

	// پیرنگ کوڈ جنریٹ کریں (یہاں آپ کے بوٹ کا نام جائے گا)
	code, err := waClient.PairPhone(context.Background(), req.Number, true, whatsmeow.PairClientChrome, "Silent Hackers")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to generate pairing code: " + err.Error()})
		return
	}

	// کامیابی کے ساتھ کوڈ واپس بھیجیں
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": code,
		"message": "Enter this code on your WhatsApp",
	})
}

// یہ ویب ساکٹ صرف فرنٹ اینڈ پر "Green Light" شو کروانے کے لیے ہے
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// ریلوے کنکشن ڈراپ ہونے سے بچانے کے لیے پنگ (Ping) لوپ
	for {
		err := conn.WriteMessage(websocket.PingMessage, nil)
		if err != nil {
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	// 1. ڈیٹا بیس سٹارٹ کریں
	db := initDatabase()
	deviceStore, err := db.GetFirstDevice()
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	// 2. واٹس ایپ کلائنٹ انیشلائز کریں
	clientLog := waLog.Stdout("Client", "INFO", true)
	waClient = whatsmeow.NewClient(deviceStore, clientLog)

	// اگر پہلے سے لاگ ان ہے تو آٹو کنیکٹ کر دیں
	if waClient.Store.ID != nil {
		fmt.Println("Already logged in, connecting...")
		waClient.Connect()
	}

	// 3. API روٹس (Routes)
	http.HandleFunc("/api/pair", pairHandler)
	http.HandleFunc("/ws", wsHandler)
	
	// فرنٹ اینڈ سرونگ (index.html اور pic.png کے لیے)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Silent Hackers Gateway running on 0.0.0.0:%s\n", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
