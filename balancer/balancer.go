package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

var nextServerIndex int = 0

func main() {
	var mu sync.Mutex

	originServerList := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	loadBalancerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Use a mutex to prevent data races when updating nextServerIndex.
		mu.Lock()

		// Select the next backend server in the pool.
		originServerURL, _ := url.Parse(originServerList[(nextServerIndex)%len(originServerList)])

		nextServerIndex++

		mu.Unlock()

		// Use an existing reverse proxy from httputil to forward the request to the selected backend server.
		reverseProxy := httputil.NewSingleHostReverseProxy(originServerURL)

		reverseProxy.ServeHTTP(rw, req)
	})

	log.Fatal(http.ListenAndServe(":8080", loadBalancerHandler))
}
