package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	env "github.com/MattAitchison/envconfig"
	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/connectable/pkg/lookup"

	_ "github.com/gliderlabs/connectable/pkg/lookup/dns"
)

var Version string

var (
	endpoint = env.String("docker_host", "unix:///var/run/docker.sock", "docker endpoint")
	port     = env.String("port", "10000", "primary listen port")

	self *docker.Container
)

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func runNetCmd(container, image string, cmd string) error {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return err
	}
	c, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:      image,
			Cmd:        []string{cmd},
			Entrypoint: []string{"/bin/sh", "-c"},
		},
		HostConfig: &docker.HostConfig{
			Privileged:  true,
			NetworkMode: fmt.Sprintf("container:%s", container),
		},
	})
	if err != nil {
		return err
	}
	if err := client.StartContainer(c.ID, nil); err != nil {
		return err
	}
	status, err := client.WaitContainer(c.ID)
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("netcmd non-zero exit: %v", status)
	}
	return client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    c.ID,
		Force: true,
	})
}

func originalDestinationPort(conn net.Conn) (string, error) {
	f, err := conn.(*net.TCPConn).File()
	if err != nil {
		return "", err
	}
	defer f.Close()
	addr, err := syscall.GetsockoptIPv6Mreq(
		int(f.Fd()), syscall.IPPROTO_IP, 80) // 80 = SO_ORIGINAL_DST
	if err != nil {
		return "", err
	}
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])
	return strconv.Itoa(int(port)), nil
}

func inspectBackend(sourceIP, destPort string) (string, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return "", err
	}
	label := fmt.Sprintf("connect[%s]", destPort)

	// todo: cache, invalidate with container destroy events
	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return "", err
	}
	for _, listing := range containers {
		container, err := client.InspectContainer(listing.ID)
		if err != nil {
			return "", err
		}
		if container.NetworkSettings.IPAddress == sourceIP {
			backend, ok := container.Config.Labels[label]
			if !ok {
				return "", fmt.Errorf("connect label '%s' not found: %v", label, container.Config.Labels)
			}
			return backend, nil
		}
	}
	return "", fmt.Errorf("unable to find container with source IP")
}

func lookupBackend(conn net.Conn) string {
	sourceIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	destPort, err := originalDestinationPort(conn)
	if err != nil {
		log.Println("unable to determine destination port")
		return ""
	}

	backend, err := inspectBackend(sourceIP, destPort)
	if err != nil {
		log.Println(err)
		return ""
	}
	return backend
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

func setupContainer(id string) error {
	re := regexp.MustCompile("connect\\[(\\d+)\\]")
	client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Println(err)
		return err
	}
	container, err := client.InspectContainer(id)
	if err != nil {
		log.Println(err)
		return err
	}
	if container.HostConfig.NetworkMode == "bridge" || container.HostConfig.NetworkMode == "default" {
		hasBackends := false
		cmds := []string{
			"/sbin/sysctl -w net.ipv4.conf.all.route_localnet=1",
			"iptables -t nat -I POSTROUTING 1 -m addrtype --src-type LOCAL --dst-type UNICAST -j MASQUERADE",
		}
		for k, _ := range container.Config.Labels {
			results := re.FindStringSubmatch(k)
			if len(results) > 1 {
				hasBackends = true
				cmds = append(cmds, fmt.Sprintf(
					"iptables -t nat -I OUTPUT 1 -m addrtype --src-type LOCAL --dst-type LOCAL -p tcp --dport %s -j DNAT --to-destination %s:%s",
					results[1], self.NetworkSettings.IPAddress, results[1]))
			}
		}
		if hasBackends {
			log.Printf("setting iptables on %s \n", container.ID[:12])
			shellCmd := strings.Join(cmds, " && ")
			err := runNetCmd(container.ID, self.Image, shellCmd)
			if err != nil {
				log.Printf("error setting iptables on %s: %s \n", container.ID[:12], err)
				return err
			}
		}
	}
	return nil

}

func monitorContainers() {
	client, err := docker.NewClient(endpoint)
	assert(err)
	events := make(chan *docker.APIEvents)
	assert(client.AddEventListener(events))
	list, _ := client.ListContainers(docker.ListContainersOptions{})
	for _, listing := range list {
		go setupContainer(listing.ID)
	}
	for msg := range events {
		switch msg.Status {
		case "create":
			go setupContainer(msg.ID)
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":"+port)
	assert(err)

	fmt.Printf("# Connectable %s listening on %s ...\n", Version, port)

	client, err := docker.NewClient(endpoint)
	assert(err)

	list, err := client.ListContainers(docker.ListContainersOptions{})
	assert(err)
	for _, listing := range list {
		c, err := client.InspectContainer(listing.ID)
		assert(err)
		if c.Config.Hostname == os.Getenv("HOSTNAME") {
			self = c
			if c.HostConfig.NetworkMode == "bridge" || c.HostConfig.NetworkMode == "default" {
				fmt.Printf("# Setting iptables on connectable... ")
				shellCmd := fmt.Sprintf("iptables -t nat -A PREROUTING -p tcp -j REDIRECT --to-ports %s", port)
				assert(runNetCmd(c.ID, c.Image, shellCmd))
				fmt.Printf("done.\n")
			}
		}
	}

	if self == nil {
		fmt.Println("# unable to find self")
		os.Exit(1)
	}

	go monitorContainers()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		backend := lookupBackend(conn)
		if backend == "" {
			conn.Close()
			continue
		}

		backendAddrs, err := lookup.Resolve(backend)
		if err != nil {
			log.Println(err)
			conn.Close()
			continue
		}
		if len(backendAddrs) == 0 {
			log.Println(conn.RemoteAddr(), backend, "no backends")
			conn.Close()
			continue
		}

		log.Println(conn.RemoteAddr(), backend, "->", backendAddrs[0])
		go proxyConn(conn, backendAddrs[0])
	}
}
