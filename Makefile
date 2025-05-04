.PHONY: all build clean install uninstall run

BINARY_NAME=gps-timesync
GO=go
ARGS=

all: build

build:
	$(GO) build -o $(BINARY_NAME) main.go

clean:
	rm -f $(BINARY_NAME)

run:
	$(GO) run main.go $(ARGS)

install: build
	install -d $(DESTDIR)/usr/bin
	install -d $(DESTDIR)/usr/share/man/man1
	install -m 755 $(BINARY_NAME) $(DESTDIR)/usr/bin/
	install -m 644 man/man1/gps-timesync.1 $(DESTDIR)/usr/share/man/man1/
	gzip -f $(DESTDIR)/usr/share/man/man1/gps-timesync.1

uninstall:
	rm -f $(DESTDIR)/usr/bin/$(BINARY_NAME)
	rm -f $(DESTDIR)/usr/share/man/man1/gps-timesync.1.gz 