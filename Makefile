.PHONY: dist clean

build: openess

DEVICE=192.168.1.37:58899

openess:
ifdef ARCH
	GOOS=linux GOARCH=$(ARCH) go build -o openess cmd/openess/*.go
else
	go build -o openess cmd/openess/*.go
endif

run-daemon: openess
	go run ./cmd/openess/*.go -b -d $(DEVICE) -l debug 

run-cli: openess
	go run ./cmd/openess/*.go -d $(DEVICE) -l debug

clean:
	go clean
	rm -f openess
	rm -f openess.tar.xz

openess.tar.xz: openess
	tar cvf openess.tar.xz openess data

dist: openess.tar.xz
