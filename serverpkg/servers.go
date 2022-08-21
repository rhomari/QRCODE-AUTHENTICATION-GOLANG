package serverpkg

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type authClient struct {
	id    string
	token string
}

var authClients = make(map[string]authClient) //map of authorized users

type Client struct {
	conn    *websocket.Conn
	qrimage []byte
}
type Session struct {
	userName string
	userdata string
}

var Clients = make(map[string]Client)
var AuthUsers = map[string]Session{ // this is a dummy session database
	"c81e8366-0d2c-42b3-8639-8cbc7373f71c": {"Tariq", "Tariq's DATA"},
	"7f42512b-0772-1283-8478-604cefef32c1": {"Misty", "Misty's DATA"},
	"5f42c12b-0892-4483-8498-504defeb32a1": {"Sky", "Sky's DATA"},
}

func StarServing(addr string) {

	http.HandleFunc("/", homeHanlder)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/authenticate", authHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/qrcode", qrcodeHandler)

	http.ListenAndServe(addr, nil)
}

func homeHanlder(w http.ResponseWriter, r *http.Request) {
	if exist, _ := getCookie(r); exist == true {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Println(r.URL)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	for {

		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		if string(p) == "makeqrcode" {

			newuuid := base64.StdEncoding.EncodeToString(MakeNewQUID())
			if err := conn.WriteMessage(messageType, []byte("QRCODEMADE:"+newuuid)); err != nil {

				log.Println(err)
				return
			}

			qrimage, err := qrcode.Encode(newuuid, qrcode.Medium, 200)
			if err != nil {
				log.Println(err)
				return
			}
			Clients[newuuid] = Client{conn, qrimage}
		}

	}
}
func MakeNewQUID() []byte {
	return []byte(uuid.New().String())

}
func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	uuid := r.URL.Query().Get("id")
	log.Println(uuid)
	if client, ok := Clients[uuid]; ok {
		w.Header().Set("Content-Type", "image/png")
		w.Write(client.qrimage)
		client.conn.WriteMessage(websocket.TextMessage, []byte("QRCODE SENT"))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
func authHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	token := r.URL.Query().Get("token")
	id := r.URL.Query().Get("id")
	log.Println(id, token)
	if client, ok := Clients[id]; ok {
		client.conn.WriteMessage(websocket.TextMessage, []byte("AUTHENTICATED:"+id))
		log.Println("AUTHENTICATED:", token)
		authClients[id] = authClient{id, token}
		defer client.conn.Close()
		delete(Clients, id)

	}

}
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Length before : ", len(authClients))
	if exist, token := getCookie(r); exist == true {
		log.Println("Auth user token:", token)
		w.Write([]byte("<h1>Welcome " + AuthUsers[token].userName + "</h1>"))
		w.Write([]byte("<h2>Your Data is: " + AuthUsers[token].userdata + "</h2>"))
		return

	}

	id := r.URL.Query().Get("id")
	log.Println("dashboardHandler id:", id)
	if authclient, ok := authClients[id]; ok {
		log.Println("Auth user token:", authclient.token)
		http.SetCookie(w, &http.Cookie{Name: "token", Value: authclient.token, HttpOnly: true, Expires: time.Now().Add(time.Hour * 24)})
		w.Write([]byte("<h1>Welcome " + AuthUsers[authclient.token].userName + "</h1>"))
		w.Write([]byte("<h2>Your Data is: " + AuthUsers[authclient.token].userdata + "</h2>"))
		delete(authClients, id)
		log.Println("Length after : ", len(authClients))
		return

	}
	http.Redirect(w, r, "/", http.StatusUnauthorized)

}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if exist, _ := getCookie(r); exist == true {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	log.Println("loginHandler :  serving login page")
	http.ServeFile(w, r, "static/login.html")

}
func getCookie(r *http.Request) (exist bool, token string) {
	tokencookie, err := r.Cookie("token")
	if err != nil {
		log.Println(err)
		return false, ""

	}
	return true, tokencookie.Value

}
