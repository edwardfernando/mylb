package lb

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

type Node struct {
	URL          *url.URL
	alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func (n *Node) IsAlive() bool {
	n.mux.Lock()
	alive := n.alive
	n.mux.Unlock()
	return alive
}

func (n *Node) SetAlive(alive bool) {
	n.mux.Lock()
	n.alive = alive
	n.mux.Unlock()
}
