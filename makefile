test:
	go test -v

coverage:
	go test -coverprofile=coverage.out && go tool cover -html=coverage.out

lint:
	$(GOPATH)/bin/golangci-lint run ./...
