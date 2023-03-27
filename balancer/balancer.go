package balancer

import (
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
	s.selectServer(w, r).ReverseProxy.ServeHTTP(w, r)
}

func (s *ServerPool) selectServer(w http.ResponseWriter, r *http.Request) *Backend {
	// this is for persistance support feature
	// where the same request coming from one node should be handled by
	// the same node for its future request
	cookie, err := r.Cookie("session")
	if err == nil {
		for _, backend := range s.Backends {
			if backend.URL.String() == cookie.Value {
				return backend
			}
		}
	}

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

	s.cookie = &http.Cookie{
		Name:  "session",
		Value: server.URL.String(),
		Path:  "/",
	}

	http.SetCookie(w, s.cookie)

	return server
}

func newServerPool(originServerList []string) (*ServerPool, error) {
	backends := make([]*Backend, 0, len(originServerList))

	for _, urlString := range originServerList {
		url, err := url.Parse(urlString)
		if err != nil {
			return nil, err
		}

		backend := &Backend{
			URL:          url,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		}

		backends = append(backends, backend)
	}

	return &ServerPool{
		Backends: backends,
	}, nil
}

func NewLoadBalancer(originServerList []string) (*http.Server, error) {
	serverPool, err := newServerPool(originServerList)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:    ":8000",
		Handler: serverPool,
	}, nil
}
