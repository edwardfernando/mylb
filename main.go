package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"mylb/lb"
)

func main() {
	data, err := ioutil.ReadFile("serverlist.json")
	if err != nil {
		panic(err)
	}

	var originServerList []string
	err = json.Unmarshal(data, &originServerList)
	if err != nil {
		panic(err)
	}

	portFlag := flag.Int("port", 8000, "listening port")
	flag.Parse()

	pool, err := lb.NewLoadBalancer(originServerList, *portFlag)
	if err != nil {
		panic(err)
	}

	log.Default().Printf("Starting server on port %d ...", *portFlag)
	log.Fatal(pool.ListenAndServe())
}
