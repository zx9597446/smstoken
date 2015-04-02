package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

const testSecret = "abcd"

func init() {
	cfg.SecretKey = "abcd"
	cfg.PreventSendInSeconds = 60
	cfg.TokenKeepAliveInSeconds = 3600
	cfg.TokenLength = 6
}

func TestRandInt(t *testing.T) {
	min := 5
	max := 10
	for i := 0; i < 100; i++ {
		n := randInt(min, max)
		if n < min || n > max {
			t.Fatal("randInt failed", n)
		}
	}
}

func TestRandomNumber(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := randomNumber(cfg.TokenLength)
		if len(s) != cfg.TokenLength {
			t.Fatal("TestRandomNumber failed")
		}
	}
}

func prepareApi(t *testing.T, f rest.HandlerFunc) http.Handler {
	api := makeApi()
	handler := api.MakeHandler()
	if handler == nil {
		t.Fatal("the http.Handler must be have been create")
	}
	return handler
}

type fakeSender struct {
	sendCount     int
	t             *testing.T
	lastSend      time.Time
	tokenLastSend string
}

func (fs *fakeSender) SendSMS(send Send) error {
	if len(send.Token) != cfg.TokenLength {
		fs.t.Fatal("no generated token")
	}
	d := time.Since(fs.lastSend)
	if fs.sendCount > 0 && int(d.Seconds()) < cfg.PreventSendInSeconds {
		fs.t.Fatal("send interval failed", d.Seconds(), fs.sendCount)
	}
	fs.lastSend = time.Now()
	fs.sendCount++
	fs.tokenLastSend = send.Token
	return nil
}

func newFakeSender(t *testing.T) *fakeSender {
	return &fakeSender{0, t, time.Now(), ""}
}

func makeSignaureHeader(secret string) string {
	timestamp := strconv.Itoa(int(time.Now().Unix()))
	signature := fmt.Sprintf("%x", md5.Sum([]byte(timestamp+secret)))
	return fmt.Sprintf("%s,%s", timestamp, signature)
}

func TestPostSendUnauthorized(t *testing.T) {
	gSMSSender = newFakeSender(t)
	handler := prepareApi(t, postSend)
	req := test.MakeSimpleRequest("POST", "http://localhost/send", nil)
	//header := makeSignaureHeader(testSecret)
	//req.Header.Set(headerKey, header)
	recorded := test.RunRequest(t, handler, req)
	recorded.CodeIs(401)
}

func TestPostSendBadRequest(t *testing.T) {
	gSMSSender = newFakeSender(t)
	handler := prepareApi(t, postSend)
	req := test.MakeSimpleRequest("POST", "http://localhost/send", nil)
	header := makeSignaureHeader(testSecret)
	req.Header.Set(headerKey, header)
	recorded := test.RunRequest(t, handler, req)
	recorded.CodeIs(400)
}

func requestSend(send Send, t *testing.T) *fakeSender {
	fake := newFakeSender(t)
	gSMSSender = fake
	handler := prepareApi(t, postSend)
	req := test.MakeSimpleRequest("POST", "http://localhost/send", send)
	header := makeSignaureHeader(testSecret)
	req.Header.Set(headerKey, header)
	recorded := test.RunRequest(t, handler, req)
	recorded.CodeIs(200)
	return fake
}

func requestValidation(phone, token string, t *testing.T) {
	handler := prepareApi(t, getValidation)
	url := fmt.Sprintf("http://localhost/validation/%s/%s", phone, token)
	req := test.MakeSimpleRequest("GET", url, "")
	header := makeSignaureHeader(testSecret)
	req.Header.Set(headerKey, header)
	recorded := test.RunRequest(t, handler, req)
	recorded.CodeIs(200)
}

func TestPostSendPassed(t *testing.T) {
	send := Send{}
	send.From = "from"
	send.To = "to"
	send.Text = "text: "
	requestSend(send, t)
}

func TestPreventSendTooMany(t *testing.T) {
	send := Send{}
	send.From = "from"
	send.To = randomNumber(4)
	send.Text = "text: "
	for i := 0; i < 10; i++ {
		requestSend(send, t)
	}
}

func TestGetValidation(t *testing.T) {
	send := Send{}
	send.From = "from"
	send.To = randomNumber(4)
	send.Text = "text: "
	fake := requestSend(send, t)
	requestValidation(send.To, fake.tokenLastSend, t)
}
