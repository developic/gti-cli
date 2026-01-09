PREFIX := /usr/local/bin
VERSION := 1.1.0  # version for builds
LDFLAGS := -X gti/src/cmd.Version=$(VERSION)

.PHONY: all
all:
	go build -ldflags "-X gti/src/cmd.Version=$(VERSION)" -o bin/gti main.go

.PHONY: install
install:
	install -d $(PREFIX)/bin
	install -d $(PREFIX)/share/man/man1
	install -m755 gti $(PREFIX)/bin
	install -m644 gti.1.gz $(PREFIX)/share/man/man1

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/gti
	rm -f $(DESTDIR)$(PREFIX)/share/man/man1/gti.1.gz

.PHONY: assets
assets:
	gzip -c gti.1 > gti.1.gz

.PHONY: rel
rel:
	# Linux 
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o gti-linux main.go
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o gti-linux_arm64 main.go

	# macOS 
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o gti-mac main.go

	# Windows x64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o gti.exe main.go
