.PHONY: build install clean

build:
	go build -o commity ./cmd/commity

install:
	go install ./cmd/commity

clean:
	rm -f commity
