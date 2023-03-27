package main

import (
	"log"
	"mylb/balancer"
)

func main() {
	// move this to viper
	originServerList := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
		"http://localhost:8084",
	}

	pool, err := balancer.NewLoadBalancer(originServerList)
	if err != nil {
		panic(err)
	}

	log.Default().Println("Starting server on port 8000....")
	log.Fatal(pool.ListenAndServe())
}
