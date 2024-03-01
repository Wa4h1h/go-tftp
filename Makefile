build:
	@echo "Building binary...."
	GOARCH=arm64 GOOS=darwin go build -o ./builds/darwin/tftp_client ./cmd/client/main.go
	GOARCH=arm64 GOOS=darwin go build -o ./builds/darwin/tftp_server ./cmd/server/main.go
	GOARCH=amd64 GOOS=linux go build -o ./builds/linux/tftp_client ./cmd/client/main.go
	GOARCH=amd64 GOOS=linux go build -o ./builds/linux/tftp_server ./cmd/server/main.go

.PHONY: clean
clean:
	@echo "Removing binaries...."
	@rm -rf builds/