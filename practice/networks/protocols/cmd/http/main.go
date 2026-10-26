package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("SERVER method=%s path=%s protocol=%s remote=%s request-id=%q", r.Method, r.URL.Path, r.Proto, r.RemoteAddr, r.Header.Get("X-Request-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]string{"message": "hello over HTTPS"})
	})
	plain := httptest.NewServer(h)
	defer plain.Close()
	log.Printf("SERVER plain HTTP listening %s", plain.URL)
	doRequest(plain.Client(), plain.URL, "plain-http")

	secure := httptest.NewUnstartedServer(h)
	secure.TLS = &tls.Config{MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}}
	secure.StartTLS()
	defer secure.Close()
	log.Printf("SERVER HTTPS = HTTP inside TLS, listening %s", secure.URL)
	client := secure.Client()
	for i := 1; i <= 2; i++ {
		doRequest(client, secure.URL, "https-keepalive")
	}
	client.CloseIdleConnections()
}

func doRequest(client *http.Client, baseURL, requestID string) {
	client.Timeout = 2 * time.Second
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/hello", nil)
	req.Header.Set("X-Request-ID", requestID)
	log.Printf("CLIENT request %s %s", req.Method, req.URL)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		log.Fatal(err)
	}
	log.Printf("CLIENT response status=%s protocol=%s body=%v", resp.Status, resp.Proto, body)
}
