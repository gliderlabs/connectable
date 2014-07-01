package main

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type BackendProvider interface {
	NextBackend(conn net.Conn) string
	String() string
}

type ConfigStore interface {
	List(path string) []string
	Get(path string) string
	Watch(path string)
}

func NewConfigStore(uri *url.URL) ConfigStore {
	factory := map[string]func(*url.URL) ConfigStore{
		"consul": NewConsulStore,
	}[uri.Scheme]
	if factory == nil {
		log.Fatal("unrecognized config store backend: ", uri.Scheme)
	}
	return factory(uri)
}

func NewBackendProvider(input string) BackendProvider {
	parts := strings.Split(input, ",")
	if len(parts) > 1 {
		return &fixedBackends{backends: parts}
	} else {
		u, _ := url.Parse(input)
		if u.Scheme != "" && u.Path != "" {
			provider := &configBackends{path: u.Path, scheme: u.Scheme, store: NewConfigStore(u)}
			go provider.WatchToUpdate()
			provider.Update()
			return provider
		} else {
			_, _, err := net.SplitHostPort(input)
			if err != nil {
				return &srvBackends{input}
			} else {
				return &fixedBackends{backends: []string{input}}
			}
		}
	}
}

type fixedBackends struct {
	backends []string
	counter  int64
}

func (b *fixedBackends) NextBackend(conn net.Conn) string {
	return b.backends[atomic.AddInt64(&b.counter, 1)%int64(len(b.backends))]
}

func (b *fixedBackends) String() string {
	return strings.Join(b.backends, ", ")
}

type srvBackends struct {
	name string
}

func (b *srvBackends) NextBackend(conn net.Conn) string {
	_, addrs, err := net.LookupSRV("", "", b.name)
	if err != nil {
		log.Println("dns:", err)
		return ""
	}
	if len(addrs) == 0 {
		return ""
	}
	port := strconv.Itoa(int(addrs[0].Port))
	return net.JoinHostPort(addrs[0].Target, port)
}

func (b *srvBackends) String() string {
	return b.name
}

type configBackends struct {
	sync.Mutex
	store    ConfigStore
	backends *fixedBackends
	path     string
	scheme   string
}

func (b *configBackends) WatchToUpdate() {
	for {
		b.store.Watch(b.path)
		b.Update()
	}
}

func (b *configBackends) Update() {
	b.Lock()
	defer b.Unlock()
	backends := b.store.List(b.path)
	if len(backends) == 0 {
		list := b.store.Get(b.path)
		backends = strings.Split(list, ",")
	}
	b.backends = &fixedBackends{backends: backends}
	log.Println("configstore:", b.backends)
}

func (b *configBackends) NextBackend(conn net.Conn) string {
	b.Lock()
	defer b.Unlock()
	return b.backends.NextBackend(conn)
}

func (b *configBackends) String() string {
	return fmt.Sprintf("%s %s", b.scheme, b.path)
}
