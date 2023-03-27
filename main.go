package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"mylb/balancer"
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

	pool, err := balancer.NewLoadBalancer(originServerList)
	if err != nil {
		panic(err)
	}

	log.Default().Println("Starting server on port 8000....")
	log.Fatal(pool.ListenAndServe())
}
