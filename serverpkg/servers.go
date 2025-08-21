package serverpkg

import (
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type authClient struct {
	id    string
	token string
}

type Client struct {
	conn    *websocket.Conn
	qrimage []byte
	wmu     sync.Mutex
}

func (c *Client) Write(mt int, p []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.conn.WriteMessage(mt, p)
}

type Session struct {
	userName string
	userdata string
}

var (
	mu          sync.RWMutex
	Clients     = make(map[string]*Client)
	authClients = make(map[string]*authClient)
	timers      = make(map[string]*time.Timer)
	AuthUsers   = map[string]*Session{
		"c81e8366-0d2c-42b3-8639-8cbc7373f71c": {"Tariq", "Tariq's DATA"},
		"7f42512b-0772-1283-8478-604cefef32c1": {"Misty", "Misty's DATA"},
		"5f42c12b-0892-4483-8498-504defeb32a1": {"Sky", "Sky's DATA"},
	}
)

func StartServing(addr string) {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/authenticate", authHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/qrcode", qrcodeHandler)
	log.Println("Listening on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if exist, _ := getCookie(r); exist {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if string(p) == "makeqrcode" {
			id := uuid.NewString()
			qrimage, err := qrcode.Encode(id, qrcode.Medium, 200)
			if err != nil {
				return
			}
			client := &Client{conn: conn, qrimage: qrimage}
			mu.Lock()
			Clients[id] = client
			if old := timers[id]; old != nil {
				old.Stop()
			}
			t := time.NewTimer(30 * time.Second)
			timers[id] = t
			mu.Unlock()
			go func(id string, t *time.Timer) {
				<-t.C
				mu.Lock()
				if c, still := Clients[id]; still {
					_ = c.Write(websocket.TextMessage, []byte("TIMEOUT"))
					_ = c.conn.Close()
					c.qrimage = nil
					delete(Clients, id)
				}
				delete(timers, id)
				mu.Unlock()
			}(id, t)
			_ = client.Write(messageType, []byte("QRCODEMADE:"+id))
			logMemUsage()
		}
	}
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	mu.RLock()
	client, ok := Clients[id]
	mu.RUnlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(client.qrimage)
	_ = client.Write(websocket.TextMessage, []byte("QRCODE SENT"))
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	id := r.URL.Query().Get("id")
	mu.Lock()
	client, ok := Clients[id]
	if ok {
		_ = client.Write(websocket.TextMessage, []byte("AUTHENTICATED:"+id))
		authClients[id] = &authClient{id: id, token: token}
		if tm := timers[id]; tm != nil {
			tm.Stop()
			delete(timers, id)
		}
		_ = client.conn.Close()
		client.qrimage = nil
		delete(Clients, id)
		logMemUsage()
	}
	mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if exist, token := getCookie(r); exist {
		mu.RLock()
		sess, ok := AuthUsers[token]
		mu.RUnlock()
		if ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<h1>Welcome " + sess.userName + "</h1>"))
			_, _ = w.Write([]byte("<h2>Your Data is: " + sess.userdata + "</h2>"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "token", Value: "", HttpOnly: true, Expires: time.Unix(0, 0)})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	id := r.URL.Query().Get("id")
	mu.Lock()
	ac, ok := authClients[id]
	if ok {
		delete(authClients, id)
	}
	mu.Unlock()
	if ok {
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    ac.token,
			HttpOnly: true,
			Expires:  time.Now().Add(24 * time.Hour),
		})
		mu.RLock()
		sess, ok := AuthUsers[ac.token]
		mu.RUnlock()
		if !ok {
			http.Error(w, "Unknown user", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<h1>Welcome " + sess.userName + "</h1>"))
		_, _ = w.Write([]byte("<h2>Your Data is: " + sess.userdata + "</h2>"))
		logMemUsage()
		return
	}
	http.Redirect(w, r, "/", http.StatusUnauthorized)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if exist, _ := getCookie(r); exist {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	http.ServeFile(w, r, "static/login.html")
}

func getCookie(r *http.Request) (exist bool, token string) {
	tokencookie, err := r.Cookie("token")
	if err != nil {
		return false, ""
	}
	return true, tokencookie.Value
}

func logMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Alloc = %v", m.Alloc)
}
