# Justfile for MailSherpa CLI

version := `git describe --tags --always`
ldflags := "-X github.com/customeros/mailsherpa/internal/cmd.version=" + version 

build-macos:
	@echo "Building MailSherpa CLI for MacOS..."
	GOOS=darwin go build -ldflags "{{ldflags}}" -o mailsherpa-macos

build-linux-amd64:
	@echo "Building MailSherpa CLI for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "{{ldflags}}" -o mailsherpa-linux-amd64

build-linux-arm64:
    @echo "Building MailSherpa CLI for Linux ARM64..."
    GOOS=linux GOARCH=arm64 go build -ldflags "{{ldflags}}" -o mailsherpa-linux-arm64

build: build-macos build-linux-amd64 build-linux-arm64

clean:
    rm -f mailsherpa-linux-arm64 mailsherpa-linux-amd64 mailsherpa-macos

test:
    go test ./...
