NAME=connectable
REPO=gliderlabs
VERSION=$(shell cat VERSION)

dev:
	@docker history $(NAME):dev &> /dev/null \
		|| docker build -f Dockerfile.dev -t $(NAME):dev .
	@docker run --rm --name $(NAME)-dev \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/go/src/github.com/$(REPO)/$(NAME) \
		$(NAME):dev
