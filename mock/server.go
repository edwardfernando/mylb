package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	portFlag := flag.Int("port", 8081, "listening port")
	flag.Parse()
	port := fmt.Sprintf(":%d", *portFlag)

	originServerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[origin server] received request: %s\n", time.Now())
		io.Copy(rw, req.Body)
	})

	log.Fatal(http.ListenAndServe(port, originServerHandler))
}
