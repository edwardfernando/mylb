package lb

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// LB represents a load balancer with the necessary configuration
type LB struct {
	Nodes   []*Node
	current int64
	mux     sync.Mutex
	cookie  *http.Cookie
}

// NewLoadBalancer creates a new load balancer with the given list of origin servers.
// It returns a new http.Server instance for the load balancer to listen on incoming requests.
func NewLoadBalancer(originServerList []string, port int) (*http.Server, error) {
	serverPool, err := newServerNodes(originServerList)
	if err != nil {
		return nil, err
	}

	go serverPool.RunHealthCheck()

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: serverPool,
	}, nil
}

// NextIndex returns the index of the next node in the slice
func (lb *LB) NextIndex() int64 {
	return atomic.AddInt64(&lb.current, int64(1)) % int64(len(lb.Nodes))
}

// ServeHTTP handles the HTTP request and sends the response back through the provided http.ResponseWriter.
// It selects a node based on the load balancing strategy
func (lb *LB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lb.mux.Lock()
	defer lb.mux.Unlock()

	node, err := lb.selectServer(w, r)
	if err != nil {
		log.Default().Fatal("failed to serve HTTP: `", err)
	}
	node.ReverseProxy.ServeHTTP(w, r)
}

// RunHealthCheck passively checks the health status of all the nodes
func (lb *LB) RunHealthCheck() {
	log.Default().Println("Running health check...")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, n := range lb.Nodes {
			status := n.CheckNode()
			n.SetAlive(status)
			statusString := "down"
			if status {
				statusString = "up"
			}

			log.Default().Printf("Node %s status: %v", n.URL.Host, statusString)
		}
	}
}

// selectServer selects a node based on the load balancing strategy
func (lb *LB) selectServer(w http.ResponseWriter, r *http.Request) (*Node, error) {
	cookie, err := r.Cookie("session")
	if err == nil {
		return lb.selectServerByCookie(w, cookie)
	}

	return lb.selectServerByNextHealthyNode(w)
}

// selectServerByCookie selects a node by session cookie
func (lb *LB) selectServerByCookie(w http.ResponseWriter, cookie *http.Cookie) (*Node, error) {
	for _, node := range lb.Nodes {
		if node.URL.String() == cookie.Value {
			if !node.IsAlive() {
				return lb.selectServerByNextHealthyNode(w)
			}

			return node, nil
		}
	}

	return lb.selectServerByNextHealthyNode(w)
}

// selectServerByNextHealthyNode selects the next healthy node
func (lb *LB) selectServerByNextHealthyNode(w http.ResponseWriter) (*Node, error) {
	node, err := lb.getNextHealthyNode()

	if err != nil {
		return nil, err
	}

	lb.setCookie(w, node)

	return node, nil
}

// getNextHealthyNode returns the next available healthy node and actively update the
// status of the choose node
func (lb *LB) getNextHealthyNode() (*Node, error) {
	for i := 0; i < len(lb.Nodes); i++ {
		node := lb.Nodes[lb.current]
		lb.current = lb.NextIndex()
		if node.IsAlive() {
			return node, nil
		} else {
			node.SetAlive(false)
		}
		log.Default().Println("Getting node ", node.URL)
		// increment the index

	}

	return nil, errors.New("no available node")
}

// setCookie sets a session cookie with the provided node URL string as value in the HTTP response writer w.
// It also sets lb.cookie to the same cookie for future reference.
func (lb *LB) setCookie(w http.ResponseWriter, node *Node) {
	lb.cookie = &http.Cookie{
		Name:  "session",
		Value: node.URL.String(),
		Path:  "/",
	}

	http.SetCookie(w, lb.cookie)
}

// newServerNodes returns a new Load Balancer (LB) struct that contains a list of Nodes,
// where each Node represents an upstream server specified in the originServerList argument.
// For each URL in originServerList, a new Node is created and appended to the nodes slice.
// The function returns an error if any URL in originServerList is invalid.
func newServerNodes(originServerList []string) (*LB, error) {
	nodes := []*Node{}

	for _, urlString := range originServerList {
		url, err := url.Parse(urlString)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(url)

		n := &Node{
			URL:          url,
			ReverseProxy: proxy,
		}

		nodes = append(nodes, n)
	}

	return &LB{
		Nodes: nodes,
		mux:   sync.Mutex{},
	}, nil
}
