package lb

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bsm/gomega"
)

func TestNextIndex(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	node1 := &Node{}
	node2 := &Node{}
	node3 := &Node{}
	lb := &LB{Nodes: []*Node{node1, node2, node3}}

	g.Expect(lb.current).To(gomega.Equal(int64(0)))
	g.Expect(lb.NextIndex()).To(gomega.Equal(int64(1)))
	g.Expect(lb.NextIndex()).To(gomega.Equal(int64(2)))
	g.Expect(lb.NextIndex()).To(gomega.Equal(int64(0)))
	g.Expect(lb.NextIndex()).To(gomega.Equal(int64(1)))
	g.Expect(lb.NextIndex()).To(gomega.Equal(int64(2)))
}

func TestRunHealthCheck(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer1.Close()

	mockServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	node1 := &Node{URL: &url.URL{Host: strings.TrimPrefix(mockServer1.URL, "http://")}, alive: true}
	node2 := &Node{URL: &url.URL{Host: strings.TrimPrefix(mockServer2.URL, "http://")}, alive: true}

	lb := &LB{Nodes: []*Node{node1, node2}}

	// Run health check
	go lb.RunHealthCheck()
	time.Sleep(10 * time.Second)

	g.Expect(node1.IsAlive()).To(gomega.BeTrue())
	g.Expect(node2.IsAlive()).To(gomega.BeTrue())

	// set node2 to be down
	mockServer2.Close()

	// Re-Run health check
	go lb.RunHealthCheck()
	time.Sleep(10 * time.Second)

	g.Expect(node1.IsAlive()).To(gomega.BeTrue())
	g.Expect(node2.IsAlive()).To(gomega.BeFalse())
}

func TestSelectServerByCookie(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	anotherUrl := url.URL{Host: "anotherurl.com"}
	cookieUrl := url.URL{Host: "example.com"}

	activeNode1 := &Node{alive: true, URL: &anotherUrl}
	activeNodeWithCookie := &Node{alive: true, URL: &cookieUrl}
	activeNode3 := &Node{alive: true, URL: &anotherUrl}

	testCases := []struct {
		name         string
		nodes        []*Node
		cookie       *http.Cookie
		expectedNode *Node
		expectedErr  error
	}{
		{
			name:         "cookie presents in the request - pick up the node that has same name with cookie",
			nodes:        []*Node{activeNode1, activeNodeWithCookie, activeNode3},
			cookie:       &http.Cookie{Value: cookieUrl.Host},
			expectedNode: activeNodeWithCookie,
			expectedErr:  nil,
		},
		{
			name:         "cookie not present in the request - use the next available healthy node",
			nodes:        []*Node{activeNode1, activeNode3},
			cookie:       &http.Cookie{Value: "random.com"},
			expectedNode: activeNode3,
			expectedErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			lb := &LB{Nodes: tc.nodes}
			node, err := lb.selectServerByCookie(w, tc.cookie)

			g.Expect(node).To(gomega.Equal(tc.expectedNode))

			if tc.expectedErr != nil {
				g.Expect(err).To(gomega.Equal(tc.expectedErr))
			} else {
				g.Expect(err).To(gomega.BeNil())
			}
		})
	}
}

func TestSelectServerByNextHealthyNode(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	url := url.URL{Host: "example.com"}

	activeNode1 := &Node{alive: true, URL: &url}
	activeNode2 := &Node{alive: true, URL: &url}

	inaactiveNode1 := &Node{alive: false, URL: &url}
	inaactiveNode2 := &Node{alive: false, URL: &url}

	testCases := []struct {
		name           string
		nodes          []*Node
		expectedNode   *Node
		expectedErr    error
		expectedCookie http.Cookie
	}{
		{
			name:           "all nodes are active",
			nodes:          []*Node{activeNode1, activeNode2},
			expectedNode:   activeNode1,
			expectedErr:    nil,
			expectedCookie: http.Cookie{Name: "session", Value: "//example.com"},
		},
		{
			name:         "all nodes are inactive",
			nodes:        []*Node{inaactiveNode1, inaactiveNode2},
			expectedNode: nil,
			expectedErr:  errors.New("no available node"),
		},
		{
			name:           "combination of active and inactive nodes",
			nodes:          []*Node{inaactiveNode1, activeNode1},
			expectedNode:   activeNode1,
			expectedErr:    nil,
			expectedCookie: http.Cookie{Name: "session", Value: "//example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lb := &LB{Nodes: tc.nodes}
			w := httptest.NewRecorder()
			node, err := lb.selectServerByNextHealthyNode(w)

			if tc.expectedErr == nil {
				cookies := w.Result().Cookies()
				cookie := cookies[0]

				g.Expect(len(cookies)).To(gomega.Equal(1))
				g.Expect(cookie.Name).To(gomega.Equal(tc.expectedCookie.Name))
				g.Expect(cookie.Value).To(gomega.Equal(tc.expectedCookie.Value))
			}

			g.Expect(node).To(gomega.Equal(tc.expectedNode))

			if tc.expectedErr != nil {
				g.Expect(err).To(gomega.Equal(tc.expectedErr))
			} else {
				g.Expect(err).To(gomega.BeNil())
			}
		})
	}
}

func TestGetNextHealthyNode(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	activeNode1 := &Node{alive: true}
	activeNode2 := &Node{alive: true}

	inaactiveNode1 := &Node{alive: false}
	inaactiveNode2 := &Node{alive: false}

	testCases := []struct {
		name         string
		nodes        []*Node
		expectedNode *Node
		expectedErr  error
	}{
		{
			name:         "all nodes are active",
			nodes:        []*Node{activeNode1, activeNode2},
			expectedNode: activeNode1,
			expectedErr:  nil,
		},
		{
			name:         "all nodes are inactive",
			nodes:        []*Node{inaactiveNode1, inaactiveNode2},
			expectedNode: nil,
			expectedErr:  errors.New("no available node"),
		},
		{
			name:         "combination of active and inactive nodes",
			nodes:        []*Node{inaactiveNode1, activeNode1},
			expectedNode: activeNode1,
			expectedErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lb := &LB{Nodes: tc.nodes}
			node, err := lb.getNextHealthyNode()

			g.Expect(node).To(gomega.Equal(tc.expectedNode))
			if tc.expectedErr != nil {
				g.Expect(err).To(gomega.Equal(tc.expectedErr))
			} else {
				g.Expect(err).To(gomega.BeNil())
			}
		})
	}
}

func TestSetCookie(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	lb := &LB{}
	node := &Node{URL: &url.URL{Host: "example.com"}}

	// Create a mock ResponseWriter
	w := httptest.NewRecorder()

	// Call setCookie function to set the cookie
	lb.setCookie(w, node)

	// Check if cookie was set
	cookies := w.Result().Cookies()
	g.Expect(len(cookies)).To(gomega.Equal(1))

	// Check cookie values
	cookie := cookies[0]
	g.Expect(cookie.Name).To(gomega.Equal("session"))

	g.Expect(cookie.Value).To(gomega.Equal("//example.com"))

	// Make a new request and add the cookie
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(cookie)

	// Check if the cookie is present in the new request
	g.Expect(len(req.Cookies())).To(gomega.Equal(1))
	g.Expect(req.Cookies()[0].Name).To(gomega.Equal("session"))
	g.Expect(req.Cookies()[0].Value).To(gomega.Equal("//example.com"))
}
