package it_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"mylb/lb"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type lbTestSuite struct {
	suite.Suite
	lb     lb.LB
	nodes  []*lb.Node
	server []*httptest.Server
}

func TestLbTestSuite(t *testing.T) {
	suite.Run(t, &lbTestSuite{})
}

func createHTTPServer(identifier string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, identifier)
	}))
	return server
}

func (l *lbTestSuite) TestLoadBalancer_AllNodesRunning() {
	servers := []string{}

	server1 := createHTTPServer("1")
	server2 := createHTTPServer("2")
	server3 := createHTTPServer("3")

	defer server1.Close()
	defer server2.Close()
	defer server3.Close()

	node1 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server1.URL, "http://")}}
	node1.SetAlive(true)

	node2 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server2.URL, "http://")}}
	node2.SetAlive(true)

	node3 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server3.URL, "http://")}}
	node3.SetAlive(true)

	servers = append(servers, server1.URL, server2.URL, server3.URL)

	// assert that all nodes are running
	for i := 1; i <= len(servers); i++ {
		client := http.Client{}
		res, err := client.Get(servers[i-1])

		l.NoError(err)
		l.Equal(http.StatusOK, res.StatusCode)

		body, _ := ioutil.ReadAll(res.Body)
		l.Assert().Equal(strconv.Itoa(i), string(body))
	}

	pool, _ := lb.NewLoadBalancer(servers, 8000)

	go pool.ListenAndServe()

	time.Sleep(5 * time.Second)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8000", nil)
	l.NoError(err)

	// test the round robin hit 1000 times
	for i := 0; i < 1000; i++ {
		client := &http.Client{}
		res, err := client.Do(req)
		l.NoError(err)
		l.Equal(http.StatusOK, res.StatusCode)

		body, _ := ioutil.ReadAll(res.Body)

		// the response should be circular between 1, 2, 3 in x times
		l.Assert().Equal(strconv.Itoa(i%len(servers)+1), string(body))
	}

	pool.Shutdown(context.TODO())
}

func (l *lbTestSuite) TestLoadBalancer_OneNodeIsDown() {
	servers := []string{}

	server1 := createHTTPServer("1")
	server2 := createHTTPServer("2")
	server3 := createHTTPServer("3")

	defer server1.Close()
	defer server2.Close()
	defer server3.Close()

	node1 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server1.URL, "http://")}}
	node1.SetAlive(true)

	node2 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server2.URL, "http://")}}
	node2.SetAlive(true)

	node3 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server3.URL, "http://")}}
	node3.SetAlive(true)

	servers = append(servers, server1.URL, server2.URL, server3.URL)

	pool, _ := lb.NewLoadBalancer(servers, 8000)
	go pool.ListenAndServe()

	time.Sleep(5 * time.Second)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8000", nil)
	l.NoError(err)

	// test the round robin hit 3 times
	for i := 0; i < 3; i++ {
		client := &http.Client{}
		res, err := client.Do(req)
		l.NoError(err)
		l.Equal(http.StatusOK, res.StatusCode)

		body, _ := ioutil.ReadAll(res.Body)

		// the response should be circular between 1, 2, 3
		l.Assert().Equal(strconv.Itoa(i%len(servers)+1), string(body))
	}

	// simulate the server1 is down
	server1.Close()

	// the next request should be handled by the server2 instead of server1
	client := &http.Client{}
	res, err := client.Do(req)
	l.NoError(err)
	l.Equal(http.StatusOK, res.StatusCode)

	body, _ := ioutil.ReadAll(res.Body)

	l.Assert().Equal("2", string(body))

	pool.Shutdown(context.TODO())
}

func (l *lbTestSuite) TestLoadBalancer_CookiePresents() {
	servers := []string{}

	server1 := createHTTPServer("1")
	server2 := createHTTPServer("2")
	server3 := createHTTPServer("3")

	defer server1.Close()
	defer server2.Close()
	defer server3.Close()

	node1 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server1.URL, "http://")}}
	node1.SetAlive(true)

	node2 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server2.URL, "http://")}}
	node2.SetAlive(true)

	node3 := lb.Node{URL: &url.URL{Host: strings.TrimPrefix(server3.URL, "http://")}}
	node3.SetAlive(true)

	servers = append(servers, server1.URL, server2.URL, server3.URL)

	pool, _ := lb.NewLoadBalancer(servers, 8000)
	go pool.ListenAndServe()

	time.Sleep(5 * time.Second)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8000", nil)
	l.NoError(err)

	// the request coming from a client with cookie that has server2's URL in it
	cookie := &http.Cookie{Name: "session", Value: server2.URL}
	req.AddCookie(cookie)

	// test the round robin hit 1000 times
	// all the request should be handled by server2
	for i := 0; i < 10; i++ {
		client := &http.Client{}
		res, err := client.Do(req)
		l.NoError(err)
		l.Equal(http.StatusOK, res.StatusCode)

		body, _ := ioutil.ReadAll(res.Body)

		// the response should be always be 2
		l.Assert().Equal("2", string(body))
	}

	// now set server2 to down
	server2.Close()

	// if we dont have cookie the next request should be handled by another running server in circular order
	// the response should be always be 1, 3, 1, 3, 1, 3 ...
	client := &http.Client{}
	res, err := client.Do(req)
	l.NoError(err)
	l.Equal(http.StatusOK, res.StatusCode)

	body, _ := ioutil.ReadAll(res.Body)
	l.Assert().Equal("1", string(body))

	client = &http.Client{}
	res, err = client.Do(req)
	l.NoError(err)
	l.Equal(http.StatusOK, res.StatusCode)

	body, _ = ioutil.ReadAll(res.Body)
	l.Assert().Equal("3", string(body))

	client = &http.Client{}
	res, err = client.Do(req)
	l.NoError(err)
	l.Equal(http.StatusOK, res.StatusCode)

	body, _ = ioutil.ReadAll(res.Body)
	l.Assert().Equal("1", string(body))

	client = &http.Client{}
	res, err = client.Do(req)
	l.NoError(err)
	l.Equal(http.StatusOK, res.StatusCode)

	body, _ = ioutil.ReadAll(res.Body)
	l.Assert().Equal("3", string(body))

	pool.Shutdown(context.TODO())
}

func (l *lbTestSuite) TestLoadBalancer_NoNodesRunning() {
	servers := []string{"server1.com", "server2.com", "server3.com"}

	pool, _ := lb.NewLoadBalancer(servers, 8000)
	go pool.ListenAndServe()

	time.Sleep(5 * time.Second)

	// hit the load balancer
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8000", nil)
	l.NoError(err)

	client := &http.Client{}
	res, err := client.Do(req)
	l.NoError(err)

	// the load balancer should return 5xx error
	l.Equal(http.StatusInternalServerError, res.StatusCode)

	body, _ := ioutil.ReadAll(res.Body)

	l.Assert().Equal("Internal Server Error\n", string(body))

	pool.Shutdown(context.TODO())
}
