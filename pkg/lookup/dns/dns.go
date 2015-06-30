package dns

import (
	"log"
	"net"
	"strconv"

	"github.com/gliderlabs/connectable/pkg/lookup"
	"github.com/miekg/dns"
)

var (
	config *dns.ClientConfig
)

func init() {
	lookup.Register("dns", new(dnsResolver))
	var err error
	config, err = dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.Fatal(err)
	}
}

type dnsResolver struct{}

func (r *dnsResolver) Lookup(addr string) ([]string, error) {
	query := new(dns.Msg)
	query.SetQuestion(dns.Fqdn(addr), dns.TypeSRV)
	query.RecursionDesired = false
	client := new(dns.Client)
	resp, _, err := client.Exchange(query, net.JoinHostPort(config.Servers[0], config.Port))
	if err != nil {
		return nil, err
	}
	if len(resp.Answer) == 0 {
		return []string{}, nil
	}
	var addrs []string
	for i, record := range resp.Answer {
		port := strconv.Itoa(int(record.(*dns.SRV).Port))
		ip := record.(*dns.SRV).Target
		if len(resp.Extra) >= i+1 {
			ip = resp.Extra[i].(*dns.A).A.String()
		}
		addrs = append(addrs, net.JoinHostPort(ip, port))
	}
	return addrs, nil
}
