package main

import (
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/fsouza/go-dockerclient"
)

const SO_ORIGINAL_DST = 80

func originalDestinationPort(conn net.Conn) (string, error) {
	f, err := conn.(*net.TCPConn).File()
	if err != nil {
		return "", err
	}
	defer f.Close()
	addr, err := syscall.GetsockoptIPv6Mreq(int(f.Fd()), syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		return "", err
	}
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])
	return strconv.Itoa(int(port)), nil
}

type omnimodeBackends struct {
	sync.Mutex
	client    *docker.Client
	providers map[string]BackendProvider
}

func NewOmniProvider() BackendProvider {
	endpoint := getopt("DOCKER_HOST", "unix:///var/run/docker.sock")
	client, err := docker.NewClient(endpoint)
	assert(err)
	return &omnimodeBackends{client: client, providers: make(map[string]BackendProvider)}
}

func (b *omnimodeBackends) String() string {
	return "omnimode"
}

func (b *omnimodeBackends) lookupProvider(sourceIP, destPort string) (BackendProvider, error) {
	b.Lock()
	defer b.Unlock()
	key := sourceIP + ":" + destPort
	provider, found := b.providers[key]
	if found {
		return provider, nil
	}
	name, err := b.inspectBackendName(sourceIP, destPort)
	if err != nil {
		return nil, err
	}
	provider = NewBackendProvider(name)
	b.providers[key] = provider
	return provider, nil
}

func (b *omnimodeBackends) inspectBackendName(sourceIP, destPort string) (string, error) {
	envKey := "BACKEND_" + destPort

	// todo: cache, invalidate with container destroy events
	containers, err := b.client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return "", errors.New("omni: unable to list containers")
	}
	for _, listing := range containers {
		container, err := b.client.InspectContainer(listing.ID)
		if err != nil {
			return "", errors.New("omni: unable to inspect container " + listing.ID)
		}
		if container.NetworkSettings.IPAddress == sourceIP {
			for _, env := range container.Config.Env {
				parts := strings.SplitN(env, "=", 2)
				if strings.ToLower(parts[0]) == strings.ToLower(envKey) {
					return parts[1], nil
				}
			}
			return "", errors.New("omni: backend not found in container environment")
		}
	}
	return "", errors.New("omni: unable to find container with source IP")
}

func (b *omnimodeBackends) NextBackend(conn net.Conn) string {
	sourceIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	destPort, err := originalDestinationPort(conn)
	if err != nil {
		log.Println("omni: unable to determine destination port")
		return ""
	}

	backends, err := b.lookupProvider(sourceIP, destPort)
	if err != nil {
		log.Println(err)
		return ""
	}
	return backends.NextBackend(conn)
}
