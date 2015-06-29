package lookup

import (
	"time"

	env "github.com/MattAitchison/envconfig"
)

var (
	resolverName = env.String("lookup_resolver", "dns", "resolver to use for lookups")
	resolvers    = make(map[string]Resolver)
)

type Resolver interface {
	Lookup(addr string) ([]string, error)
}

func Register(name string, resolver Resolver) {
	resolvers[name] = resolver
}

func Resolve(addr string) ([]string, error) {
	cached, ok := resolveCache.Get(addr)
	if ok && !cached.(*cacheValue).Expired() {
		return cached.(*cacheValue).Value, nil
	}
	resolver, ok := resolvers[resolverName]
	if !ok {
		return []string{}, nil
	}
	value, err := resolver.Lookup(addr)
	if err != nil {
		return nil, err
	}
	resolveCache.Set(addr, &cacheValue{value, time.Now().Unix()})
	return value, nil
}
