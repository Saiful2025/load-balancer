package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL     string
	Proxy   *httputil.ReverseProxy
	Healthy bool
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64
}

func NewLoadBalancer(urls []string) (*LoadBalancer, error) {
	backends := make([]*Backend, len(urls))
	for i, u := range urls {
		parsedURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %s: %w", u, err)
		}
		backends[i] = &Backend{
			URL:     u,
			Proxy:   httputil.NewSingleHostReverseProxy(parsedURL),
			Healthy: true,
		}
	}
	return &LoadBalancer{backends: backends}, nil
}

func (lb *LoadBalancer) getNextBackend() *Backend {
	for i := 0; i < len(lb.backends); i++ {
		idx := atomic.AddUint64(&lb.current, 1) % uint64(len(lb.backends))
		backend := lb.backends[idx]
		if backend.Healthy {
			return backend
		}
	}
	return nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.getNextBackend()
	if backend == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	fmt.Printf("Routing to %s\n", backend.URL)
	backend.Proxy.ServeHTTP(w, r)
}

func main() {
	lb, err := NewLoadBalancer([]string{
		"http://localhost:9001",
		"http://localhost:9002",
	})
	if err != nil {
		log.Fatalf("Failed to create load balancer: %v", err)
	}
	http.ListenAndServe(":8080", lb)
}
