package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	getAddress() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type SimpleServer struct {
	Address string
	Proxy   *httputil.ReverseProxy
}

// Implement Server Interface methods on SimpleServer
func (server *SimpleServer) getAddress() string {
	return server.Address
}

func (server *SimpleServer) IsAlive() bool {
	return true
}

func (server *SimpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	server.Proxy.ServeHTTP(rw, req)
}

func newServer(address string) *SimpleServer {
	serverUrl, err := url.Parse(address)
	handleError("Error while parsing: %v", err)

	return &SimpleServer{
		Address: address,
		Proxy:   httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

// Load Balancer
type LoadBalancer struct {
	port            string
	servers         []Server
	roundRobinCount int
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

//func (server *SimpleServer) getAddress() string {
//	return server.Address
//}

func handleError(message string, err error) {
	if err != nil {
		fmt.Printf(message, err)
		os.Exit(1)
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwading request to address: %q\n", targetServer.getAddress())
	targetServer.Serve(rw, req)
}

func main() {
	servers := []Server{
		newServer("https://www.facebook.com/"),
		newServer("https://www.bing.com/"),
		newServer("https://www.duckduckgo.com"),
	}
	lb := newLoadBalancer("8080", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
