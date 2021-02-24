/* RabbitMQ handling for gopenqa */

package gopenqa

import (
	"encoding/json"
	"strings"

	"github.com/streadway/amqp"
)

// JobStatus is the returns struct for job status updates from RabbitMQ
type JobStatus struct {
	Arch string `json:"ARCH"`
	Build string `json:"BUILD"`
	Flavor string `json:"FLAVOR"`
	Machine string `json:"MACHINE"`
	Test string  `json:"TEST"`
	BugRef string `json:"bugref"`
	GroupID int `json:"group_id"`
	ID int `json:"id"`
	NewBuild string `json:"newbuild"`
	Reason string `json:"reason"`
	Remaining int `json:"remaining"`
	Result string `json:"result"`
}

// RabbitMQ struct is the object which handles the connection to a RabbitMQ instance
type RabbitMQ struct {
	remote string
	con    *amqp.Connection
}

// Close connection
func (mq *RabbitMQ) Close() {
	mq.con.Close()
}

// RabbitMQSubscription handles a single subscription
type RabbitMQSubscription struct {
	channel *amqp.Channel
	key     string
	obs     <-chan amqp.Delivery
}

// Receive receives a raw RabbitMQ messages
func (sub *RabbitMQSubscription) Receive() (amqp.Delivery, error) {
	return <-sub.obs, nil
}

// ReceiveJob receives the next message and try to parse it as job
func (sub *RabbitMQSubscription) ReceiveJob() (Job, error) {
	var job Job
	d, err := sub.Receive()
	if err != nil {
		return job, err
	}
	// Try to unmarshall to json
	err = json.Unmarshal(d.Body, &job)
	if err != nil {
		return job, err
	}
	// Fix missing job state on job state listener
	if strings.HasSuffix(d.RoutingKey, ".job.done") && job.State == "" {
		job.State = "done"
	}

	return job, err
}

// ReceiveJobStatus receives the next message and try to parse it as JobStatus. Use this for job status updates
func (sub *RabbitMQSubscription) ReceiveJobStatus() (JobStatus, error) {
	var status JobStatus
	d, err := sub.Receive()
	if err != nil {
		return status, err
	}
	// Try to unmarshall to json
	err = json.Unmarshal(d.Body, &status)
	if err != nil {
		return status, err
	}
	return status, err
}

// Close subscription channel
func (sub *RabbitMQSubscription) Close() {
	sub.channel.Close()
}

// ConnectRabbitMQ connects to a RabbitMQ instance and returns the RabbitMQ object
func ConnectRabbitMQ(remote string) (RabbitMQ, error) {
	var err error
	rmq := RabbitMQ{remote: remote}

	rmq.con, err = amqp.Dial(remote)
	if err != nil {
		return rmq, err
	}

	return rmq, nil
}

// Subscribe to a given key and get the messages via the callback function.
// This method will return after establishing the channel and call the callback function when a new message arrives
// This message returns a RabbitMQSubscription object, which in turn can be used to receive the incoming messages
func (mq *RabbitMQ) Subscribe(key string) (RabbitMQSubscription, error) {
	var sub RabbitMQSubscription
	ch, err := mq.con.Channel()
	if err != nil {
		return sub, err
	}

	// Create message queue and receive channel
	q, err := ch.QueueDeclare("", false, false, false, false, nil)
	if err != nil {
		ch.Close()
		return sub, err
	}
	if err := ch.QueueBind(q.Name, key, "pubsub", false, nil); err != nil {
		ch.Close()
		return sub, err
	}
	obs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		return sub, err
	}
	sub.channel = ch
	sub.key = key
	sub.obs = obs
	return sub, nil
}
