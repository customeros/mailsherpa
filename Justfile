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

clean:
    rm -f mailsherpa-*

dist:
	just build-macos
	tar -czpf mailsherpa-macos.tar.gz mailsherpa-macos
	aws s3 cp ./mailsherpa-macos.tar.gz $CLOUDFLARE_R2_BUCKET/mailsherpa-macos.tar.gz --endpoint-url $CLOUDFLARE_R2_ENDPOINT

	just build-linux-amd64
	tar -czpf mailsherpa-linux-amd64.tar.gz mailsherpa-linux-amd64
	aws s3 cp ./mailsherpa-linux-amd64.tar.gz $CLOUDFLARE_R2_BUCKET/mailsherpa-linux-amd64.tar.gz --endpoint-url $CLOUDFLARE_R2_ENDPOINT

	just build-linux-arm64
	tar -czpf mailsherpa-linux-arm64.tar.gz mailsherpa-linux-arm64
	aws s3 cp ./mailsherpa-linux-arm64.tar.gz $CLOUDFLARE_R2_BUCKET/mailsherpa-linux-arm64.tar.gz --endpoint-url $CLOUDFLARE_R2_ENDPOINT
	just clean

	aws s3 cp ./install.py $CLOUDFLARE_R2_BUCKET/install.py --endpoint-url $CLOUDFLARE_R2_ENDPOINT

test:
    go test ./...
