package balancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	Backends []*Backend
	current  int64
	mux      sync.Mutex
	cookie   *http.Cookie
}

func (s *ServerPool) NextIndex() int64 {
	return atomic.AddInt64(&s.current, int64(1)) % int64(len(s.Backends))
}

func (s *ServerPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// this is for persistance support feature
	// where the same request coming from one node should be handled by
	// the same node for its future request
	// if s.cookie != nil {
	// 	for _, server := range s.Backends {
	// 		if server.URL.String() == s.cookie.Value {
	// 			fmt.Println("disini")
	// 			server.ReverseProxy.ServeHTTP(w, r)
	// 			return
	// 		}
	// 	}
	// }

	s.current = s.NextIndex()
	server := s.Backends[s.current]

	if !server.Alive {
		// If the selected server is unhealthy, try the other servers until
		// a healthy one is found
		for i := 0; i < len(s.Backends); i++ {
			s.current = s.NextIndex()
			server := s.Backends[s.current]
			if server.Alive {
				break
			}
		}
	}

	fmt.Println("server: ", server.URL.Host)

	// s.cookie = &http.Cookie{
	// 	Name:  "session",
	// 	Value: server.URL.String(),
	// 	Path:  "/",
	// }

	// http.SetCookie(w, s.cookie)
	server.ReverseProxy.ServeHTTP(w, r)
}
