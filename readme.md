# MyLB

This project is a personal project aimed at designing and implementing a basic load balancer using the Go programming language. The primary objective of this undertaking is to gain a deeper understanding of how load balancers work, and to acquire hands-on experience in building one from the ground up.

Currently there are three key features:
1. Round robin load balancing strategy
2. Active & Passive health check of all nodes
3. Session affinity


## Prerequisites
1. go 1.19
2. makefile (mac: https://formulae.brew.sh/formula/make, ubuntu: https://linuxhint.com/install-make-ubuntu/)

## Usage

To use the load balancer, you must first create a new load balancer instance by providing a list of origin servers and a port number to listen for incoming requests:

```golang
// pass all of your server address 
lbServer, err := lb.NewLoadBalancer([]string{"http://localhost:8081", "http://localhost:8082"}, 8888)

if err != nil {
    log.Fatal(err)
}

log.Fatal(lbServer.ListenAndServe())
```

## Experiment
To experiment with the features, you can use the built-in mocking server and load balancer by running the available command in the Makefile.

1. Four backend servers are available, which you can spin up by running the following commands in separate terminal windows. Each command starts an independent server on ports 8081 to 8084.
```golang
  cd /path/to/this/repository
  make run.mock.server.1

  cd /path/to/this/repository
  make run.mock.server.2

  cd /path/to/this/repository
  make run.mock.server.3

  cd /path/to/this/repository
  make run.mock.server.4
```

2. In another terminal window, run the load balancer. This command starts the load balancer on port 8000.
```golang
  cd /path/to/this/repository
  make run.load.balancer
```

3. Make multiple requests to localhost:8000, and you will see that the requests are evenly distributed among the backend nodes. You can this simple `curl` command:

```golang
  curl -i localhost:8000
```



## Health Check
By default, the load balancer conducts a health check every 5 seconds to verify the status of all nodes. If a node is found to be down, it will be marked as such and the load balancer will discontinue routing traffic to it. Additionally, an active health check feature is in place whereby if a request arrives and the selected node is down, the traffic will be automatically redirected to another available node.

## Load Balancing
The load balancer uses a round-robin load balancing strategy to distribute traffic among the available nodes.

## Session Affinity
The load balancer supports session affinity by setting a session cookie with the value of the selected node URL. The cookie is stored in the HTTP response writer, and the same cookie is used for subsequent requests from the same client. If the selected node is down, the load balancer will choose the next available healthy node.