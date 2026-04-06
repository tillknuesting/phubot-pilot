.PHONY: all build clean install

all: build

build:
	go build -ldflags "-X main.version=$$(git rev-parse --short HEAD 2>/dev/null || echo dev)" -o phubot-pilot .

clean:
	rm -f phubot-pilot

install: build
	sudo cp phubot-pilot /usr/local/bin/
	sudo cp phubot-pilot.yaml /etc/phubot-pilot.yaml
