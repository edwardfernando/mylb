package lb

import (
	"net"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Node represents a server node with its URL, alive status, reverse proxy, and a mutex for synchronization.
type Node struct {
	URL          *url.URL
	alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// IsAlive returns whether the node is currently marked as alive.
// This method uses a read-write mutex to ensure that it's thread-safe.
func (n *Node) IsAlive() bool {
	n.mux.Lock()
	alive := n.alive
	n.mux.Unlock()
	return alive
}

// SetAlive sets the node's alive status to the given value.
// It uses a RWMutex to ensure safe concurrent access to the node's alive status.
func (n *Node) SetAlive(alive bool) {
	n.mux.Lock()
	n.alive = alive
	n.mux.Unlock()
}

// CheckNode checks the availability of the node by attempting to establish
// a TCP connection to its URL. Returns true if successful, false otherwise.
func (n *Node) CheckNode() bool {
	timeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", n.URL.Host, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
