# Connectable (previously ambassadord)

A smart Docker proxy that lets your containers connect to other containers via service
discovery without being service discovery aware.

## Getting Connectable

You can get the Connectable micro container from the Docker Hub.

	$ docker pull progrium/connectable

## Using Connectable

Basic overview is:

 1. Run a service registry like Consul, perhaps with Registrator
 1. Start a Connectable container on each host
 1. Expose Connectable to your containers, using links or Resolvable (experimental)
 1. Run containers with environment variables defining what they need to connect to
 1. Have software in those containers connect to backing services via Connectable

#### Starting Connectable

Once you have a service registry set up, you point Connectable to it when you launch it.
You also need to mount the Docker socket. Here is an example using the local Consul agent, assuming you're running Resolvable:

	$ docker run -d --name connectable \
			-v /var/run/docker.sock:/var/run/docker.sock \
			progrium/connectable:latest \
			consul://consul.docker:8500

Connectable supports other registries and is extendable via modules.

If you're running Consul with DNS available to your containers using the `-dns` flag or using Resolvable, you don't have to specify any registry. Connectable will do SRV lookups via DNS by default:

	$ docker run -d --name connectable \
			-v /var/run/docker.sock:/var/run/docker.sock \
			progrium/connectable:latest

#### Start containers that use Connectable

First, you need to expose the running Connectable container to new containers. Normally this is done using links. Links are unnecessary if you're running Resolvable. However, in this example we'll use links.

The only other step you need to do is specify the name of the service(s) you want to connect to using environment variables. You also specify a port you'll use when connecting to Connectable. For example:

	CONNECT_6000=redis.service.consul

With this environment variable set, you can connect to Connectable on 6000 and it will take you to the Redis service found via Consul DNS. You can also specify multiple services:

	$ docker run -d --name myservice \
			-e CONNECT_6000=redis.service.consul \
			-e CONNECT_3306=master.mysql.service.consul \
			--link connectable:connectable \
			example/myservice

Now in your service container, you can connect to `connectable:6000` to get to Redis and `connectable:3306` to get to your MySQL master. If you were using Resolvable, you could drop the link and use `connectable.docker` instead of `connectable` when connecting.

## Configuring Connectable

Connectable just needs to know what service registry to use and how to use it. It's easiest when you have DNS discovery available and you don't need to specify a registry. But, for example, if you were using etcd, you'd need to specify a URI with the etcd address and the prefix used for your services:

	$ docker run -d --name connectable \
			-v /var/run/docker.sock:/var/run/docker.sock \
			progrium/connectable:latest \
			etcd://etcd.docker:4001/path/to/services

Different registry modules have different options that can be passed in the URI. For example, there is a `consulkv` module to use the KV store instead of Consul's service discovery API.

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

The original ambassadord proof of concept was made possible thanks to [DigitalOcean](http://digitalocean.com). Also thanks to [Jérôme Petazzoni](https://github.com/jpetazzo) for helping with the bits that make this magical.

## License

BSD
