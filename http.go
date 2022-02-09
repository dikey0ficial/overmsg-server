package main

import (
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func mw(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
			} else if r.Method == "OPTIONS" {
				w.WriteHeader(405)
				fmt.Fprint(w, "Unsupported method")
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func root(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<!DOCTYPE html>"+
		"<html><head><title>OVERMSG server</title></head>"+
		"<body><h1 align=\"center\">OVERMSG server</h1><hr/>"+
		"<h2 align=\"center\">Some info:</h2>"+
		"<p> - <b>User-Agent:</b> "+r.Header.Get("User-Agent")+"</p>"+
		"<p> - <b>Method:</b> "+r.Method+"</p>"+
		func() string {
			if _, ok := r.URL.Query()["no-time"]; ok {
				return ""
			}
			return "<p> - <b>Time of response (unix):</b> <span id=\"time\">" + strconv.Itoa(int(time.Now().Unix())) + "</span></p>"
		}(),
		"<p align=center>This server is for API of simple (and unsafe) messenger, written on Go — OVERMSG</p>"+
			"</body></html>",
	)
}

const allowedSymbols = "QWERTYUIOPASDFGHJKLZXCVBNM" +
	"qwertyuiopasdfghjklzxcvbnm" +
	"0123456789" +
	"_-"

// RegHandler handles registration
func RegHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported method", nil}.ToJSON())
		return
	} else if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(415)
		w.Write(Answer{false, "Unsupported Content-Type", nil}.ToJSON())
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	if len([]rune(strings.TrimSpace(string(body)))) == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no data", nil}.ToJSON())
		return

	}
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(400)
		w.Write(Answer{false, "Invalid JSON data", nil}.ToJSON())
		return
	}
	var name, pass string
	if elem, ok := req["name"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `Got no "name" field`, nil}.ToJSON())
		return
	} else if name, ok = elem.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `"name" field is not string type`, nil}.ToJSON())
		return
	} else if name = strings.TrimSpace(name); len([]rune(name)) <= 3 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Too short name", nil}.ToJSON())
		return
	} else if len([]rune(name)) > 32 {
		w.WriteHeader(413)
		w.Write(Answer{false, "Too long name", nil}.ToJSON())
		return
	} else if []rune(name)[0] == '_' {
		w.WriteHeader(400)
		w.Write(Answer{false, "Name shouldn't start with _", nil}.ToJSON())
		return
	}
	var isValid bool = true
BIG:
	for _, sym := range []rune(name) {
		for _, vs := range []rune(allowedSymbols) {
			if sym == vs {
				continue BIG
			}
		}
		isValid = false
		break
	}
	if !isValid {
		w.WriteHeader(400)
		w.Write(Answer{false, "Name contains not-allowed symbols. GET /allowed_syms to more ingo", nil}.ToJSON())
		return
	}
	if elem, ok := req["pass"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `Got no "pass" field`, nil}.ToJSON())
		return
	} else if pass, ok = elem.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `"pass" field is not string type`, nil}.ToJSON())
		return
	} else if len([]rune(pass)) == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "pass should be longer", nil}.ToJSON())
		return
	} else if len([]rune(pass)) >= 32 {
		w.WriteHeader(413)
		w.Write(Answer{false, "pass is TOO long", nil}.ToJSON())
		return
	}
	if rCount, err := loginData.CountDocuments(ctx,
		bson.M{"name": name}); err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	} else if rCount != 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Found users with this name", nil}.ToJSON())
		return
	}
	token := uuid.New().String()
	var newUser = User{
		Name:  name,
		Pass:  pass,
		Token: token,
	}
	_, err = loginData.InsertOne(ctx, newUser)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	var ans = Answer{
		Success: true,
		Res:     TokenResult{token},
	}
	w.WriteHeader(201)
	w.Write(ans.ToJSON())
}

// AllowSymsHandler handles allowed symbols list
func AllowSymsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(allowedSymbols))
}

// GetTokenHandler handles getting token by nick and password
func GetTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported method", nil}.ToJSON())
		return
	} else if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(415)
		w.Write(Answer{false, "Unsupported Content-Type", nil}.ToJSON())
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	defer r.Body.Close()
	if len([]rune(strings.TrimSpace(string(body)))) == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no data", nil}.ToJSON())
		return

	}
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(400)
		w.Write(Answer{false, "Invalid JSON data", nil}.ToJSON())
		return
	}
	var name, pass string
	if nel, ok := req["name"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no name", nil}.ToJSON())
		return
	} else if name, ok = nel.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got not-string name", nil}.ToJSON())
		return
	}
	if pel, ok := req["pass"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no pass", nil}.ToJSON())
		return
	} else if pass, ok = pel.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got not-string pass", nil}.ToJSON())
		return
	}
	if c, err := loginData.CountDocuments(ctx, bson.M{"name": name}); err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	} else if c == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Found no user with this nickname", nil}.ToJSON())
		return
	}
	var us User
	if err := loginData.FindOne(ctx, bson.M{"name": name}).Decode(&us); err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	if us.Pass != pass {
		w.WriteHeader(400)
		w.Write(Answer{false, "Wrong password", nil}.ToJSON())
		return
	}
	w.WriteHeader(200)
	w.Write(Answer{true, "", TokenResult{us.Token}}.ToJSON())
}

// SendMessageHandler handles message sending
func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported method", nil}.ToJSON())
		return
	} else if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported Content-Type", nil}.ToJSON())
		return
	}
	token := strings.TrimSpace(r.Header.Get("Auth-Token"))
	if token == "" {
		w.WriteHeader(401)
		w.Write(Answer{false, "Got no Auth-Token", nil}.ToJSON())
		return
	} else if !isValidUUID(token) {
		w.WriteHeader(400)
		w.Write(Answer{false, "Auth-Token is not valid", nil}.ToJSON())
		return
	} else if is, err := isInUsers(token); err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	} else if !is {
		w.WriteHeader(400)
		w.Write(Answer{false, "User with this token not found", nil}.ToJSON())
		return
	}
	var from User
	err := loginData.FindOne(ctx, bson.M{"_id": token}).Decode(&from)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	} else if len(strings.TrimSpace(string(data))) == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no data", nil}.ToJSON())
		return
	}
	var req SendMessageRequest
	err = json.Unmarshal(data, &req)
	if err != nil {
		w.WriteHeader(400)
		w.Write(Answer{false, "Invalid JSON", nil}.ToJSON())
		return
	}
	if strings.TrimSpace(req.PeerName) == "" {
		w.WriteHeader(400)
		w.Write(Answer{false, "Empty peer_name", nil}.ToJSON())
		return
	} else if strings.TrimSpace(req.Message) == "" {
		w.WriteHeader(400)
		w.Write(Answer{false, "Empty message", nil}.ToJSON())
		return
	} else if len([]rune(req.Message)) > 1024 {
		w.WriteHeader(413)
		w.Write(Answer{false, "Too long Message", nil}.ToJSON())
		return
	}
	if rCount, err := loginData.CountDocuments(ctx,
		bson.M{"name": req.PeerName}); err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	} else if rCount == 0 {
		w.WriteHeader(404)
		w.Write(Answer{false, "User with this name not found", nil}.ToJSON())
		return
	}
	c, ok := conns[req.PeerName]
	if !ok {
		w.WriteHeader(410)
		w.Write(Answer{false, "User is offline", nil}.ToJSON())
		return
	}
	c.Write(Message{from.Name, req.Message, "", ""}.ToJSON())
	w.WriteHeader(200)
	w.Write(Answer{true, "", nil}.ToJSON())
}

// GoOfflineHandler handles going offline
func GoOfflineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported method", nil}.ToJSON())
		return
	}
	token := strings.TrimSpace(r.Header.Get("Auth-Token"))
	if len(token) == 0 {
		w.WriteHeader(400)
		w.Write(Answer{false, "Got no Auth-Token", nil}.ToJSON())
		return
	}
	var (
		conn net.Conn
		ok   bool
	)
	if conn, ok = conns[token]; !ok {
		w.WriteHeader(404)
		w.Write(Answer{false, "Connection with this token not found", nil}.ToJSON())
		return
	}
	conn.Close()
	delete(conns, token)
	w.WriteHeader(200)
	w.Write(Answer{true, "", nil}.ToJSON())
}

// IsOnlineHandler returns is user by nick online
func IsOnlineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(405)
		w.Write(Answer{false, "Unsupported method", nil}.ToJSON())
		return
	}
	dat, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		errl.Println(err)
		w.Write(Answer{false, "Server-side error", nil}.ToJSON())
		return
	}
	defer r.Body.Close()
	var req map[string]interface{}
	if err := json.Unmarshal(dat, &req); err != nil {
		w.WriteHeader(400)
		w.Write(Answer{false, "Invalid JSON data", nil}.ToJSON())
		return
	}
	var (
		elem interface{}
		ok   bool
	)
	if elem, ok = req["name"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `Got no "name" field`, nil}.ToJSON())
		return
	}
	var name string
	if name, ok = elem.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `"name" field is not string type`, nil}.ToJSON())
		return
	}
	_, cOk := conns[name]
	w.WriteHeader(200)
	w.Write(Answer{true, "", IsOnlineResult{cOk}}.ToJSON())
}
