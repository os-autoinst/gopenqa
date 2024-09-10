package main

import (
	"fmt"
	"os"

	"github.com/os-autoinst/gopenqa"
)

func main() {
	// Create RabbitMQ instance
	rmq, err := gopenqa.ConnectRabbitMQ("amqps://opensuse:opensuse@rabbit.opensuse.org")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	defer rmq.Close()

	// Subscribe to job updates, those come in via the topic 'opensuse.openqa.job.done'
	sub, err := rmq.Subscribe("opensuse.openqa.job.done")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	defer sub.Close()
	fmt.Fprintf(os.Stderr, "Connected and subscribed to rabbit.opensuse.org\n")
	// Receive job updates, this could also happen as background thread
	for {
		if job, err := sub.ReceiveJob(); err == nil {
			fmt.Printf("%s - %s\n", job.String(), job.JobState())
		}
	}
}
