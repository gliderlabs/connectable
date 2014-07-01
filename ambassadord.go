package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func getopt(name, def string) string {
	if env := os.Getenv(name); env != "" {
		return env
	}
	return def
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func proxyConn(conn net.Conn, addr string) {
	backend, err := net.Dial("tcp", addr)
	defer conn.Close()
	if err != nil {
		log.Println("proxy", err.Error())
		return
	}
	defer backend.Close()

	done := make(chan struct{})
	go func() {
		io.Copy(backend, conn)
		backend.(*net.TCPConn).CloseWrite()
		close(done)
	}()
	io.Copy(conn, backend)
	conn.(*net.TCPConn).CloseWrite()
	<-done
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %v [backends]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	port := getopt("PORT", "10000")

	var backends BackendProvider
	if flag.Arg(0) != "" {
		backends = NewBackendProvider(flag.Arg(0))
	} else {
		backends = NewOmniProvider()
	}

	listener, err := net.Listen("tcp", ":"+port)
	assert(err)

	log.Println("Ambassadord listening on", port, "using", backends, "...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		backend := backends.NextBackend(conn)
		if backend == "" {
			conn.Close()
			log.Printf("No backends! Closing the connection")
			continue
		}

		log.Println(conn.RemoteAddr(), "->", backend)
		go proxyConn(conn, backend)
	}
}
