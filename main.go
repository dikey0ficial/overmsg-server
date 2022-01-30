package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var (
	infl, errl *log.Logger
	conf       = struct {
		Mongo struct {
			URL string `toml:"url" env:"MONGOURL,notEmpty"`
		} `toml:"mongo"`
		HTTP struct {
			Port uint16 `toml:"port" env:"HTTPPORT"`
		} `toml:"http"`
		TCP struct {
			Port uint16 `toml:"port" env:"TCPPORT"`
		} `toml:"tcp"`
	}{}

	mongoClient *mongo.Client
	appDB       *mongo.Database
	loginData   *mongo.Collection
	json             = jsoniter.ConfigCompatibleWithStandardLibrary
	ctx              = context.Background()
	conns            = make(map[string]net.Conn)
	initFailed  bool = true
)

func init() {
	fmt.Print("Open loggers: ... ")
	infl = log.New(os.Stdout, "[INFO]\t", log.Ldate|log.Ltime|log.Lshortfile)
	errl = log.New(os.Stderr, "[ERROR]\t", log.Ldate|log.Ltime|log.LstdFlags|log.Lshortfile)
	fmt.Println("\rOpen loggers: success")
	fmt.Print("Read conf: ...")
	if tomlData, err := ioutil.ReadFile("config.toml"); err == nil {
		if err := toml.Unmarshal(tomlData, &conf); err != nil {
			errl.Println(err)
			return
		}
	} else if !os.IsNotExist(err) {
		errl.Println(err)
		return
	} else if err := env.Parse(&conf); err != nil {
		errl.Println(err)
		return
	}
	fmt.Println("\rRead conf: success!")
	if conf.Mongo.URL == "" {
		errl.Println("mongo.url is empty")
		return
	}
	if conf.HTTP.Port == 0 {
		conf.HTTP.Port = 4422
	}
	if conf.TCP.Port == 0 {
		conf.TCP.Port = 4242
	}
	if conf.HTTP.Port == conf.TCP.Port {
		infl.Println("[ERROR] http.port equals tcp.port \n" +
			"(cannot use the same port for both connections)")
		return
	}
	fmt.Println("\rRead conf: success")
	fmt.Print("Open MongoDB Client: ...")
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(conf.Mongo.URL))
	if err != nil {
		errl.Println(err)
		return
	}
	fmt.Println("\rOpen MongoDB Client: success")
	fmt.Print("Connect to MongoDB: ...")
	err = mongoClient.Connect(ctx)
	if err != nil {
		errl.Println(err)
		return
	}
	fmt.Println("\rConnect to MongoDB: success")
	fmt.Print("Init MongoDB: ...")
	appDB = mongoClient.Database("app")
	loginData = appDB.Collection("login")
	fmt.Println("\rInit MongoDB: success")
	initFailed = false
}

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
		"<p> - <b>Time of response (unix):</b> <span id=\"time\">"+strconv.Itoa(int(time.Now().Unix()))+"</span></p>"+
		"<p align=center>This server is for API of simple (and unsafe) messenger, written on Go — OVERMSG</p>"+
		"</body></html>",
	)
}

// AuthHandler handles auth
func AuthHandler(w http.ResponseWriter, r *http.Request) {
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
	var (
		elem interface{}
		ok   bool
	)
	if elem, ok = req["name"]; !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `"name" field is empty`, nil}.ToJSON())
		return
	}
	var name string
	if name, ok = elem.(string); !ok {
		w.WriteHeader(400)
		w.Write(Answer{false, `"name" field is not string type`, nil}.ToJSON())
		return
	} else if len([]rune(name)) > 32 {
		w.WriteHeader(413)
		w.Write(Answer{false, "Too long name", nil}.ToJSON())
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
		Token: token,
		Name:  name,
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
		Res:     AuthResult{token},
	}
	w.WriteHeader(201)
	w.Write(ans.ToJSON())
}

// SendMessageHandler handles
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
	c.Write(Message{from.Name, req.Message, ""}.ToJSON())
	w.WriteHeader(200)
	w.Write(Answer{true, "", nil}.ToJSON())
}

// GoOfflineHandler _
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

func main() {
	if initFailed {
		return
	}
	defer infl.Println("[END]   ========================")
	router := mux.NewRouter()
	router.HandleFunc("/auth", AuthHandler)
	router.HandleFunc("/go_offline", GoOfflineHandler)
	router.HandleFunc("/send_message", SendMessageHandler)
	router.HandleFunc("/", root)
	infl.Println("[START] ========================")
	var mainDeathChan = make(chan struct{})
	go func() {
		err := listenPort(conf.TCP.Port)
		if err != nil {
			errl.Println("listen tcp:", err.Error())
		}
		mainDeathChan <- struct{}{}
		return
	}()
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", conf.HTTP.Port),
			mw(router))
		if err != nil {
			errl.Println("listen http:" + err.Error())
			mainDeathChan <- struct{}{}
			return
		}
	}()
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	for {
		select {
		case <-interruptChan:
			return
		case <-mainDeathChan:
			return
		}
	}
}
