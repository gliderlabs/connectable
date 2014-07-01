FROM ubuntu:trusty

RUN apt-get update && apt-get install -y iptables socat

ADD ./stage/ambassadord /bin/ambassadord
ADD ./stage/start /start

EXPOSE 10000

ENTRYPOINT ["/start"]