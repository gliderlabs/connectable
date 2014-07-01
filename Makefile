
build:
	go build -o stage/ambassadord
	docker build -t ambassadord .

.PHONY: build