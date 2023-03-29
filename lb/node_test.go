package lb

import (
	"fmt"
	"strings"
	"time"

	"github.com/bsm/gomega"

	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestIsAlive(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name          string
		isAlive       bool
		expectedValue bool
	}{
		{
			name:          "isAlive set to true",
			isAlive:       true,
			expectedValue: true,
		},
		{
			name:          "isAlive set to false",
			isAlive:       false,
			expectedValue: false,
		},
		{
			name:          "isAlive empty",
			expectedValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &Node{
				URL:   &url.URL{Host: "localhost:8000"},
				alive: tc.isAlive,
			}

			g.Expect(node.IsAlive()).To(gomega.Equal(tc.expectedValue))
		})
	}
}

func TestSetAlive(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name          string
		setAlive      bool
		expectedValue bool
	}{
		{
			name:          "setAlive to true",
			setAlive:      true,
			expectedValue: true,
		},
		{
			name:          "setAlive to false",
			setAlive:      false,
			expectedValue: false,
		},
		{
			name:          "setAlive not called",
			expectedValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			node := &Node{
				URL: &url.URL{Host: "localhost:8000"},
			}

			node.SetAlive(tc.setAlive)

			g.Expect(node.IsAlive()).To(gomega.Equal(tc.expectedValue))
		})
	}
}

func TestCheckNode(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name          string
		expectedValue bool
	}{
		{
			name:          "node is running",
			expectedValue: true,
		},
		{
			name:          "no is not running",
			expectedValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := &url.URL{}

			if tc.expectedValue {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintln(w, "Hello, world!")
				})

				testServer := httptest.NewServer(handler)
				defer testServer.Close()

				url.Host = strings.TrimPrefix(testServer.URL, "http://")
			}

			node := &Node{
				URL: url,
			}

			g.Expect(node.CheckNode()).To(gomega.Equal(tc.expectedValue))

		})
	}

}

func TestCheckResponseTime(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name              string
		timeout           time.Duration
		expectedUnhealthy bool
		expectedWeight    float64
	}{
		{
			name:              "node unhealthy",
			timeout:           time.Duration(500 * time.Millisecond),
			expectedUnhealthy: true,
			expectedWeight:    0.9,
		},
		{
			name:              "node healthy",
			timeout:           0,
			expectedUnhealthy: false,
			expectedWeight:    1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := &url.URL{}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.timeout > 0 {
					time.Sleep(tc.timeout)
				}
			})

			testServer := httptest.NewServer(handler)
			defer testServer.Close()

			url.Host = strings.TrimPrefix(testServer.URL, "http://")

			node := &Node{
				URL:    url,
				weight: 1,
			}
			fmt.Println(node.weight)
			node.CheckResponseTime()
			fmt.Println(node.weight)

			g.Expect(tc.expectedUnhealthy).To(gomega.Equal(node.unhealthy))
			g.Expect(tc.expectedWeight).To(gomega.Equal(node.weight))
		})
	}
}
