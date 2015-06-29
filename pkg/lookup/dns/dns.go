package dns

import (
	"net"
	"strconv"

	"github.com/progrium/connectable/pkg/lookup"
)

func init() {
	lookup.Register("dns", new(dnsResolver))
}

type dnsResolver struct{}

func (r *dnsResolver) Lookup(addr string) ([]string, error) {
	_, records, err := net.LookupSRV("", "", addr)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []string{}, nil
	}
	var addrs []string
	for _, record := range records {
		port := strconv.Itoa(int(record.Port))
		addrs = append(addrs, net.JoinHostPort(record.Target, port))
	}
	return addrs, nil
}
