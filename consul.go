package main

import (
	"log"
	"net/url"

	"github.com/armon/consul-api"
)

type ConsulStore struct {
	client    *consulapi.Client
	waitIndex uint64
}

func NewConsulStore(uri *url.URL) ConfigStore {
	config := consulapi.DefaultConfig()
	if uri.Host != "" {
		config.Address = uri.Host
	}
	client, err := consulapi.NewClient(config)
	assert(err)
	return &ConsulStore{client: client}
}

func (s *ConsulStore) List(path string) []string {
	kv, _, err := s.client.KV().List(path, &consulapi.QueryOptions{})
	if err != nil {
		log.Println("consul:", err)
		return []string{}
	}
	list := make([]string, 0)
	for _, pair := range kv {
		list = append(list, string(pair.Value))
	}
	return list
}

func (s *ConsulStore) Get(path string) string {
	kv, _, err := s.client.KV().Get(path, &consulapi.QueryOptions{})
	if err != nil {
		log.Println("consul:", err)
		return ""
	}
	if kv == nil {
		return ""
	}
	return string(kv.Value)
}

func (s *ConsulStore) Watch(path string) error {
	_, meta, err := s.client.KV().Get(path, &consulapi.QueryOptions{WaitIndex: s.waitIndex})
	if err != nil {
		log.Println("consul:", err)
	} else {
		s.waitIndex = meta.LastIndex
	}
	return err
}
