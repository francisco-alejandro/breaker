test:
	go test -v `go list ./... | grep -v example`
	
coverage:
	go test -v `go list ./... | grep -v example` -coverprofile=coverage.out && go tool cover -html=coverage.out

lint:
	$(GOPATH)/bin/golangci-lint run ./...
