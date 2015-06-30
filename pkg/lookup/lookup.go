package lookup

import (
	"fmt"
	"log"
	"time"

	env "github.com/MattAitchison/envconfig"
)

var (
	resolverName = env.String("lookup_resolver", "dns", "resolver to use for lookups")
	debugMode    = env.Bool("lookup_debug", false, "enable debug output")
	resolvers    = make(map[string]Resolver)
)

func debug(v ...interface{}) {
	if debugMode {
		log.Println(v...)
	}
}

type Resolver interface {
	Lookup(addr string) ([]string, error)
}

func Register(name string, resolver Resolver) {
	resolvers[name] = resolver
}

func Resolve(addr string) ([]string, error) {
	cached, ok := resolveCache.Get(addr)
	if ok && !cached.(*cacheValue).Expired() {
		debug("lookup: resolving [cache]:", addr, cached.(*cacheValue).Value)
		return cached.(*cacheValue).Value, nil
	}
	resolver, ok := resolvers[resolverName]
	if !ok {
		debug("lookup: resolver not found:", resolverName)
		return []string{}, fmt.Errorf("resolver not found: %s", resolverName)
	}
	value, err := resolver.Lookup(addr)
	if err != nil {
		return nil, err
	}
	resolveCache.Set(addr, &cacheValue{value, time.Now().Unix()})
	debug("lookup: resolving:", addr, value)
	return value, nil
}
