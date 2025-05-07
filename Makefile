.PHONY: all build clean install uninstall run build-simulator run-simulator

BINARY_NAME=gps-timesync
SIMULATOR_DIR=gps-simulator
SIMULATOR_BINARY_NAME=gps-simulator
GO=go
ARGS=
SIM_ARGS=

all: build build-simulator

build:
	$(GO) build -o $(BINARY_NAME) main.go

build-simulator:
	cd $(SIMULATOR_DIR) && $(GO) build -o $(SIMULATOR_BINARY_NAME) simulator.go

clean:
	rm -f $(BINARY_NAME)
	rm -f $(SIMULATOR_DIR)/$(SIMULATOR_BINARY_NAME)

run:
	$(GO) run main.go $(ARGS)

run-simulator:
	cd $(SIMULATOR_DIR) && $(GO) run simulator.go $(SIM_ARGS)

install: build
	install -d $(DESTDIR)/usr/bin
	install -d $(DESTDIR)/usr/share/man/man1
	install -m 755 $(BINARY_NAME) $(DESTDIR)/usr/bin/
	install -m 644 man/man1/gps-timesync.1 $(DESTDIR)/usr/share/man/man1/
	gzip -f $(DESTDIR)/usr/share/man/man1/gps-timesync.1

uninstall:
	rm -f $(DESTDIR)/usr/bin/$(BINARY_NAME)
	rm -f $(DESTDIR)/usr/share/man/man1/gps-timesync.1.gz 