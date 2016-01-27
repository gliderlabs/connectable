# Connectable

[![Docker Hub](https://img.shields.io/badge/docker-ready-blue.svg)](https://registry.hub.docker.com/u/gliderlabs/connectable/)
[![IRC Channel](https://img.shields.io/badge/irc-%23gliderlabs-blue.svg)](https://kiwiirc.com/client/irc.freenode.net/#gliderlabs)

A smart Docker proxy that lets your containers connect to other containers via service
discovery *without being service discovery aware*.

## Getting Connectable

You can get the Connectable micro container from the Docker Hub.

	$ docker pull gliderlabs/connectable

## Using Connectable

Basic overview is:

 1. Run a service registry like Consul, perhaps with Registrator
 1. Start a Connectable container on each host
 1. Expose Connectable to your containers, using links or Resolvable (experimental)
 1. Run containers with labels defining what they need to connect to with what ports
 1. Have software in those containers connect via those ports on localhost

#### Starting Connectable

Once you have a service registry set up, you point Connectable to it when you launch it.
You also need to mount the Docker socket. Here is an example using the local Consul agent, assuming you're running Resolvable:

	$ docker run -d --name connectable \
			-v /var/run/docker.sock:/var/run/docker.sock \
			gliderlabs/connectable:latest

With Resolvable running, it will have access to Consul DNS. It will be able to resolve any connections using DNS names.

#### Start containers that use Connectable

All you have to do is specify a port to use and what you'd like to connect to as a label. For example:

	connect.6000=redis.service.consul

With this label set, you can connect to Redis on localhost:6000. You can also specify multiple services:

	$ docker run -d --name myservice \
			-l connect.6000=redis.service.consul \
			-l connect.3306=master.mysql.service.consul \
			example/myservice

## Load Balancing

Connectable acts as a load balancer across instance of services it finds. It shuffles them randomly on new connections. Although this seems less predictable, it ensures even balancing cluster-wide.

Connectable is a reverse proxy and balancer, but it is not recommended to be used as your public facing balancer. Instead, use a more configurable balancer like haproxy or Nginx. Use Connectable for internal service-to-service connections. For example, you could use Connectable *with* Nginx to simplify your Nginx container setup.

## Health Checking

Currently Connectable does not have native health checking integration. For now, Connectable defers to the registry to return healthy services. For example, this is how Consul DNS works. Otherwise, when Connectable tries to connect to an endpoint and is unable to connect, it will try the next one transparently until all services have been tried. This covers some but not all "unhealthy" service cases.

Future modules may add support for integration with health checking mechanisms.

## Overhead

Like all proxies, you incur overhead to your connections. Connectable is roughly comparable but slightly slower than Nginx. Not by much. Here is some data collected using HTTP requests via ApacheBench using `-n 200 -c 20`:
```
nginx:

    Requests per second:    754.53 [#/sec] (mean)
    Time per request:       26.507 [ms] (mean)
    Time per request:       1.325 [ms] (mean, across all concurrent requests)

connectable:

    Requests per second:    606.32 [#/sec] (mean)
    Time per request:       32.986 [ms] (mean)
    Time per request:       1.649 [ms] (mean, across all concurrent requests)
```
Memory overhead is also roughly comparable per connection. Added network latency is near zero since it's running on the same host as clients. Keep in mind, Connectable is designed to run on each host for best performance and to avoid SPOF.

Although Connectable is Good Enough for most cases, if the overhead is a deal breaker for a particular case, don't use it in that particular case. Alternatives include working with service registries directly, just using DNS discovery with known ports, setting up a full SDN, etc.

## Modules

Todo

## Why not just DNS?

If you're using Consul DNS, SkyDNS, et al, you may wonder why Connectable is necessary. The answer is ports. Most software is not designed for dynamic ports. Most software can only resolve hostnames to IPs. You have to hard configure the port used.

If you are able to run all containers publishing exposed ports on known ports (`-p 80:80`), you might not need Connectable. If you have a fancy SDN solution that makes private container IPs publicly addressable and they use known ports, you don't need Connectable.

However, if you run containers with non-conventional ports, or don't have control over published ports, or just want to not care and wish it were magically taken care of ... that's what Connectable is for.

Connectable when combined with Registrator lets you run containers with `-P` and not care about what port they publish as.

Also, DNS may not randomize results, effectively balancing services. Connectable ensures internal load balancing.

## Notes

https://github.com/docker/docker/issues/7468
https://github.com/docker/docker/issues/7467

## Sponsor and Thanks

Connectable is sponsored by [Weave](http://weave.works). The original ambassadord proof of concept was made possible thanks to [DigitalOcean](http://digitalocean.com). Also thanks to [Jérôme Petazzoni](https://github.com/jpetazzo) for helping with the iptables bits that make this magical.

## License

MIT
<img src="https://ga-beacon.appspot.com/UA-58928488-2/connectable/readme?pixel" />
