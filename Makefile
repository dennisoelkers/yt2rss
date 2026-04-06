.PHONY: build clean run

build:
	go build -o yt2rss .

clean:
	rm -f yt2rss

run: build
	./yt2rss -cache-dir ./cache

.DEFAULT_GOAL := build
