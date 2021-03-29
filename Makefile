default: gopenqa
gopenqa: cmd/gopenqa/gopenqa.go *.go
	go build -o gopenqa cmd/gopenqa/gopenqa.go
