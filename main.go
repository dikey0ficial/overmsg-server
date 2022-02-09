package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/caarlos0/env"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

var (
	infl, errl, debl *log.Logger
	conf             = struct {
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
	debl = log.New(os.Stderr, "[DEBUG]\t", log.Ldate|log.Ltime|log.LstdFlags|log.Lshortfile)
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

func main() {
	if initFailed {
		return
	}
	defer infl.Println("[END]   ========================")
	router := mux.NewRouter()
	router.HandleFunc("/reg", RegHandler)
	router.HandleFunc("/get_token", GetTokenHandler)
	router.HandleFunc("/go_offline", GoOfflineHandler)
	router.HandleFunc("/send_message", SendMessageHandler)
	router.HandleFunc("/is_online", IsOnlineHandler)
	router.HandleFunc("/allowed_syms", AllowSymsHandler)
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
