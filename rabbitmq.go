/* RabbitMQ handling for gopenqa */

package gopenqa

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/streadway/amqp"
)

// JobStatus is the returns struct for job status updates from RabbitMQ
type JobStatus struct {
	Type      string      // Type of the update. Currently "job.done" and "job.restarted" are set
	Arch      string      `json:"ARCH"`
	Build     string      `json:"BUILD"`
	Flavor    string      `json:"FLAVOR"`
	Machine   string      `json:"MACHINE"`
	Test      string      `json:"TEST"`
	BugRef    string      `json:"bugref"`
	GroupID   int         `json:"group_id"`
	ID        int64       `json:"id"`
	NewBuild  string      `json:"newbuild"`
	Reason    string      `json:"reason"`
	Remaining int         `json:"remaining"`
	Result    interface{} `json:"result"`
}

// RabbitMQ comment
type CommentMQ struct {
	ID      int    `json:"id"`
	Created string `json:"created"`
	Updates string `json:"updated"`
	Text    string `json:"text"`
	User    string `json:"user"`
}

// RabbitMQ struct is the object which handles the connection to a RabbitMQ instance
type RabbitMQ struct {
	remote string
	con    *amqp.Connection
	closed bool
}

// Callback when the connection was closed
type RabbitMQCloseCallback func(error)

// Close connection
func (mq *RabbitMQ) Close() {
	mq.closed = true
	mq.con.Close()
}

// Connected returns true if RabbitMQ is connected
func (mq *RabbitMQ) Connected() bool {
	return !mq.closed && !mq.con.IsClosed()
}

// Connected returns true if RabbitMQ is closing or if it is closed.
func (mq *RabbitMQ) Closed() bool {
	if mq.closed {
		return true
	}
	if mq.con.IsClosed() {
		mq.closed = true
		return true
	}
	return false
}

// Reconnect to the RabbitMQ server. This will close any previous connections and channels
func (mq *RabbitMQ) Reconnect() error {
	var err error
	mq.con.Close()
	mq.closed = false
	mq.con, err = amqp.Dial(mq.remote)
	return err
}

// NotifyClose registeres a defined callback function for when the RabbitMQ connection is closed
func (mq *RabbitMQ) NotifyClose(callback RabbitMQCloseCallback) {
	go func() {
		recvChannel := make(chan *amqp.Error, 1)
		mq.con.NotifyClose(recvChannel)
		for err := range recvChannel {
			callback(fmt.Errorf(err.Error()))
		}
	}()
}

// RabbitMQSubscription handles a single subscription
type RabbitMQSubscription struct {
	channel *amqp.Channel
	key     string
	obs     <-chan amqp.Delivery
	mq      *RabbitMQ
	con     *amqp.Connection // Keep a reference to the connection to check if it is still connected. This is necessary because mq can reconnect and therefore have another new mq.con instance
}

// Connected returns true if RabbitMQ is connected
func (sub *RabbitMQSubscription) Connected() bool {
	return !sub.con.IsClosed()
}

// Receive receives a raw non-empty RabbitMQ messages
func (sub *RabbitMQSubscription) Receive() (amqp.Delivery, error) {
	for msg, ok := <-sub.obs; ok; {
		if len(msg.Body) > 0 {
			return msg, nil
		}
	}
	if sub.mq == nil || sub.mq.closed || sub.con == nil || sub.con.IsClosed() {
		return amqp.Delivery{}, fmt.Errorf("EOF")
	}
	return amqp.Delivery{}, fmt.Errorf("channel unexpectedly closed")
}

// ReceiveChannel returns the subscription's read channel
func (sub *RabbitMQSubscription) ReceiveChannel() <-chan amqp.Delivery {
	return sub.obs
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

	type IJobStatus struct {
		Type      string      // Type of the update. Currently "job.done" and "job.restarted" are set
		Arch      string      `json:"ARCH"`
		Build     string      `json:"BUILD"`
		Flavor    string      `json:"FLAVOR"`
		Machine   string      `json:"MACHINE"`
		Test      string      `json:"TEST"`
		BugRef    string      `json:"bugref"`
		GroupID   int         `json:"group_id"`
		ID        interface{} `json:"id"`
		NewBuild  string      `json:"newbuild"`
		Reason    string      `json:"reason"`
		Remaining int         `json:"remaining"`
		Result    interface{} `json:"result"`
	}
	// Try to unmarshall to json
	var istatus IJobStatus
	err = json.Unmarshal(d.Body, &istatus)
	if err != nil {
		return status, err
	}
	status.Arch = istatus.Arch
	status.Build = istatus.Build
	status.Flavor = istatus.Flavor
	status.Machine = istatus.Machine
	status.Test = istatus.Test
	status.BugRef = istatus.BugRef
	status.GroupID = istatus.GroupID
	status.NewBuild = istatus.NewBuild
	status.Reason = istatus.Reason
	status.Remaining = istatus.Remaining
	status.Result = istatus.Result

	// Due to poo#114529 we need to do a bit of magic with the ID
	if unboxed, ok := istatus.ID.(string); ok {
		status.ID, _ = strconv.ParseInt(unboxed, 10, 64) // ignore error
	} else if unboxed, ok := istatus.ID.(int64); ok {
		status.ID = unboxed
	} else if unboxed, ok := istatus.ID.(int); ok {
		status.ID = int64(unboxed)
	} else if unboxed, ok := istatus.ID.(float64); ok {
		// Values larger than int are sometimes parsed as float64
		status.ID = int64(float64(unboxed))
	} else {
		return status, fmt.Errorf("invalid ID type")
	}

	// Determine type based on routing key
	key := d.RoutingKey
	if strings.HasSuffix(key, ".job.done") {
		status.Type = "job.done"
	} else if strings.HasSuffix(key, ".job.restart") {
		status.Type = "job.restarted"
	}
	return status, nil
}

// ReceiveJobStatus receives the next message and try to parse it as Comment. Use this for job status updates
func (sub *RabbitMQSubscription) ReceiveComment() (CommentMQ, error) {
	var comment CommentMQ
	d, err := sub.Receive()
	if err != nil {
		return comment, err
	}
	// Try to unmarshall to json
	err = json.Unmarshal(d.Body, &comment)
	if err != nil {
		return comment, err
	}
	return comment, err
}

// Close subscription channel
func (sub *RabbitMQSubscription) Close() {
	sub.channel.Close()
}

// ConnectRabbitMQ connects to a RabbitMQ instance and returns the RabbitMQ object
func ConnectRabbitMQ(remote string) (RabbitMQ, error) {
	var err error
	rmq := RabbitMQ{remote: remote, closed: false}

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
	// Create a new exclusive queue with auto-delete
	q, err := ch.QueueDeclare("", false, false, true, true, nil)
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
	sub.con = mq.con
	return sub, nil
}
