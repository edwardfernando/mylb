package lb

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type LB struct {
	Nodes   []*Node
	current int64
	mux     sync.Mutex
	cookie  *http.Cookie
}

func (lb *LB) NextIndex() int64 {
	return atomic.AddInt64(&lb.current, int64(1)) % int64(len(lb.Nodes))
}

func (lb *LB) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lb.mux.Lock()
	defer lb.mux.Unlock()
	lb.selectServer(w, r).ReverseProxy.ServeHTTP(w, r)
}

func (lb *LB) selectServer(w http.ResponseWriter, r *http.Request) *Node {
	// this is for persistance support feature
	// where the same request coming from one node should be handled by
	// the same node for its future request
	cookie, err := r.Cookie("session")
	if err == nil {
		for _, s := range lb.Nodes {
			if s.URL.String() == cookie.Value {
				return s
			}
		}
	}

	lb.current = lb.NextIndex()
	node := lb.Nodes[lb.current]

	if !node.IsAlive() {
		// If the selected server is unhealthy, try the other servers until
		// a healthy one is found
		for i := 0; i < len(lb.Nodes); i++ {
			lb.current = lb.NextIndex()
			node := lb.Nodes[lb.current]
			if node.IsAlive() {
				break
			}
		}
	}

	lb.cookie = &http.Cookie{
		Name:  "session",
		Value: node.URL.String(),
		Path:  "/",
	}

	http.SetCookie(w, lb.cookie)

	return node
}

func newServerNodes(originServerList []string) (*LB, error) {
	nodes := []*Node{}

	for _, urlString := range originServerList {
		url, err := url.Parse(urlString)
		if err != nil {
			return nil, err
		}

		n := &Node{
			URL:          url,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		}

		nodes = append(nodes, n)
	}

	return &LB{
		Nodes: nodes,
	}, nil
}

func NewLoadBalancer(originServerList []string, port int) (*http.Server, error) {
	serverPool, err := newServerNodes(originServerList)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: serverPool,
	}, nil
}
