package main

import (
	"log"
	"mylb/balancer"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {

	originServerList := []string{
		"localhost:8081",
		"localhost:8082",
		"localhost:8083",
		"localhost:8084",
	}

	var backends []*balancer.Backend
	for _, urlString := range originServerList {

		url := &url.URL{
			Scheme: "http",
			Host:   urlString,
		}

		backend := &balancer.Backend{
			URL:          url,
			Alive:        true, // todo: add checking for this
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		}

		backends = append(backends, backend)
	}

	serverPool := &balancer.ServerPool{Backends: backends}

	server := &http.Server{
		Addr:    ":8000",
		Handler: serverPool,
	}

	log.Fatal(server.ListenAndServe())
}
