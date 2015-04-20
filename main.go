package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/hoisie/redis"
	"github.com/sfreiberg/gotwilio"
	"github.com/zx9597446/conf"
	"github.com/zx9597446/sigchecker"
)

var redisClt redis.Client

const headerKey = "X-Sms-Signature"

var cfgFile = flag.String("c", "config.conf", "config file in json")
var addr = flag.String("addr", ":8080", "address to listen")
var redisAddr = flag.String("redis", "127.0.0.1:6379", "redis address")

type Config struct {
	SecretKey               string
	TwilioId                string
	TwilioKey               string
	TokenKeepAliveInSeconds int
	PreventSendInSeconds    int
	TokenLength             int
}

var cfg Config

type Send struct {
	From       string
	To         string
	Text       string
	Token      string    //auto generated
	LastSendAt time.Time //auto generated
}

type SMSSender interface {
	SendSMS(Send) error
}

var gSMSSender SMSSender

type TwilioSender struct {
	client *gotwilio.Twilio
}

func NewTwilioSender() *TwilioSender {
	return &TwilioSender{gotwilio.NewTwilioClient(cfg.TwilioId, cfg.TwilioKey)}
}

func (ts *TwilioSender) SendSMS(send Send) error {
	log.Println(send.From, send.To, send.Text+send.Token)
	resp, except, err := ts.client.SendSMS(send.From, send.To, send.Text+send.Token, "", "")
	log.Println(resp, except)
	return err
}

func sendSMS(send Send) {
	if err := gSMSSender.SendSMS(send); err != nil {
		log.Fatalln(err)
	}
}

func saveToRedis(send Send, isExpire bool) {
	send.LastSendAt = time.Now()
	stream, _ := json.Marshal(send)
	if isExpire {
		redisClt.Setex(send.To, int64(cfg.TokenKeepAliveInSeconds), stream)
	} else {
		redisClt.Set(send.To, stream)
	}
}

func postSend(w rest.ResponseWriter, r *rest.Request) {
	send := Send{}
	if err := r.DecodeJsonPayload(&send); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	bs, err := redisClt.Get(send.To)
	if err != nil {
		send.Token = randomNumber(cfg.TokenLength)
		sendSMS(send)
		saveToRedis(send, true)
		w.WriteHeader(http.StatusOK)
		return
	}
	saved := Send{}
	err = json.Unmarshal(bs, &saved)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d := time.Since(saved.LastSendAt)
	if int(d.Seconds()) <= cfg.PreventSendInSeconds {
		w.WriteHeader(http.StatusOK)
		return
	}
	sendSMS(saved)
	saveToRedis(saved, false)
	w.WriteHeader(http.StatusOK)
}

func getValidation(w rest.ResponseWriter, r *rest.Request) {
	phone := r.PathParam("phone")
	valid := r.PathParam("token")
	log.Println(valid, len(valid), cfg.TokenLength)
	if len(valid) != cfg.TokenLength {
		rest.Error(w, "bad parameter", http.StatusBadRequest)
		return
	}
	bs, err := redisClt.Get(phone)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	saved := Send{}
	err = json.Unmarshal(bs, &saved)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result := (saved.Token == valid)
	w.WriteJson(map[string]bool{"result": result})
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	redisClt.Addr = *redisAddr
}

func makeApi() *rest.Api {
	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	api.Use(sigchecker.NewSignatureChecker(headerKey, cfg.SecretKey))
	router, err := rest.MakeRouter(
		&rest.Route{"GET", "/validation/:phone/:token", getValidation},
		&rest.Route{"POST", "/send", postSend},
	)
	if err != nil {
		log.Fatal(err)
	}
	api.SetApp(router)
	return api
}

func main() {
	flag.Parse()
	if err := conf.Load(*cfgFile, &cfg); err != nil {
		return
	}
	gSMSSender = NewTwilioSender()
	api := makeApi()
	log.Fatal(http.ListenAndServe(*addr, api.MakeHandler()))
}

func randomNumber(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(48, 57))
	}
	return string(bytes)
}
func randInt(min int, max int) int {
	var bytes int
	bytes = min + rand.Intn(max-min)
	return int(bytes)
}
