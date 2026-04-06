.PHONY: all build clean install

all: build

build:
	go build -o phubot-pilot .

clean:
	rm -f phubot-pilot

install: build
	sudo cp phubot-pilot /usr/local/bin/
	sudo cp phubot-pilot.yaml /etc/phubot-pilot.yaml
