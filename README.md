> NOTE: This project is starting to get some attention from mentions at conferences and such, but it's important to know that this project is likely to change drastically (perhaps even change its name) very soon. Consider it a proof of concept. 

# ambassadord

A Docker ambassador (containerized TCP reverse proxy / forwarder) that supports static forwards, DNS-based forwards (with SRV), Consul+Etcd based forwards, or forwards based on the connecting container's intended backend (read: magic).

## Getting ambassadord

You can get the ambassadord container from the Docker Hub. It is a trusted build.

	$ docker pull progrium/ambassadord

## Using ambassadord

There are two ways to use ambassadord. The first is called standard mode, where it acts as an ambassador for one pre-defined type of backend. This is most like the normal ambassador pattern, but with support for dynamic backend lookups. 

The second is called omni mode, where it can be used for *any* type of backend based on data provided by the connecting container's environment. Using omni mode means you only need to run one ambassador on a host, then all containers can use it for connecting to all their dependent backends. 

### Standard mode (aka boring-but-useful mode)

Ambassador to fixed/resolved backend(s) using domain or IP and port. Listens on same port as destination.

	$ docker run -d --expose 8080 progrium/ambassadord 192.168.1.100:8080
	$ docker run -d --expose 6379 progrium/ambassadord redis.example.com:6379
	$ docker run -d --expose 6379 progrium/ambassadord redis-1.example.com:6379,redis-2.example.com:6379

Ambassador to fixed backend defined by container link(s). Listens on the same ports as the link ports.

	$ docker run -d --expose 6379 --link redis:redis progrium/ambassadord --links
	$ docker run -d --expose 6379 --expose 8080 --link redis:redis --link http:http progrium/ambassadord --links

Ambassador to backends resolved using SRV from DNS (ie Consul service discovery). Always listens on default exposed port 10000.

	$ docker run -d progrium/ambassadord redis.service.consul

Ambassador to backends found in configuration KV store. Uses the values of child nodes if the path has children, otherwise, uses the value found at the given path. Lookups are cached and updated when the value changes in the configuration store. Always listens on exposed port 10000.

	$ docker run -d progrium/ambassadord etcd://127.0.0.1:4001/path/to/backend/nodes
	$ docker run -d progrium/ambassadord consul://127.0.0.1:8500/path/to/backend/nodes

### Omni mode (aka magic mode)

Start the ambassador in omni mode, passing the Docker socket:

	$ docker run -d -v /var/run/docker.sock:/var/run/docker.sock --name backends progrium/ambassadord --omnimode

This container, named `backends`, listens on exported port 10000, but needs to handle connections on all ports on its interface. To configure this, we run another ambassadord container with `--privileged` attached to the `backends` container's network, and we tell it to set up iptables for the ambassadord container:

	$ docker run --rm --privileged --net container:backends progrium/ambassadord --setup-iptables

Now we can start other containers that use ambassadord. They must be linked to the ambassadord container and specify their outgoing connection backends via `BACKEND` environment variables. For example, here we start a container that will have outgoing connections on 6379 sent to the backend defined by DNS SRV records for `redis.services.consul`:

	$ docker run -d --link backends:backends -e "BACKEND_6379=redis.services.consul" progrium/mycontainer startdaemon

Inside this container, any connections to `backends:6379` will be forwarded to backends resolved by `redis.services.consul`. You can set up multiple backends for several ports by adding more `BACKEND` environment variables using the port you'll be connecting with in the name and the backend to use in the value. Any of the backend definitions supported by standard mode can be used as a value.

## Sponsor and Thanks

This project was made possible thanks to [DigitalOcean](http://digitalocean.com). Also thanks to [Jérôme Petazzoni](https://github.com/jpetazzo) for helping me make the magic of omni mode work.

## License

BSD
