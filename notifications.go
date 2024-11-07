package go_sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/threatwinds/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var notificationsChannel chan *Message

// Represent the details of a success or failure during the processing of a log. Used as a message body for notifications.
type NotificationMessage struct {
	Cause      *string `json:"cause,omitempty"`
	DataType   string  `json:"dataType"`
	DataSource string  `json:"dataSource"`
}

const (
	TOPIC_ENQUEUE_FAILURE = "enqueue_failure" // TOPIC_ENQUEUE_FAILURE represents the topic name for enqueue failure notifications.
	TOPIC_ENQUEUE_SUCCESS = "enqueue_success" // TOPIC_ENQUEUE_SUCCESS represents the topic name for enqueue success notifications.
)

// SendNotificationsFromChannel listens to the notificationsChannel and sends notifications
// to the engine server via gRPC. It logs errors if the connection to the engine server fails,
// if sending a notification fails, or if receiving an acknowledgment fails. It runs indefinitely
// and should be run as a goroutine.
//
// Returns:
//
//	*logger.Error: An error object if any error occurs during the process.
func SendNotificationsFromChannel() *logger.Error {
	conn, err := grpc.NewClient(fmt.Sprintf("unix://%s", path.Join(
		GetCfg().Env.Workdir, "sockets", "engine_server.sock")),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return Logger().ErrorF("failed to connect to engine server: %v", err)
	}

	client := NewEngineClient(conn)

	notifyClient, err := client.Notify(context.Background())
	if err != nil {
		return Logger().ErrorF("failed to create notify client: %v", err)
	}

	for {
		msg := <-notificationsChannel

		err = notifyClient.Send(msg)
		if err != nil {
			return Logger().ErrorF("failed to send notification: %v", err)
		}

		ack, err := notifyClient.Recv()
		if err != nil {
			return Logger().ErrorF("failed to receive notification ack: %v", err)
		}

		Logger().LogF(100, "received notification ack: %v", ack)
	}
}


// EnqueueNotification sends a notification message to a specified topic.
// It marshals the NotificationMessage into JSON format and sends it to the notifications channel.
//
// Parameters:
//   - topic: The topic to which the notification message will be sent.
//   - body: The NotificationMessage to be sent.
//
// Returns:
//   - *logger.Error: Returns an error if the message marshalling fails, otherwise returns nil.
func EnqueueNotification(topic string, body NotificationMessage) *logger.Error {
	mByte, err := json.Marshal(body)
	if err != nil {
		return Logger().ErrorF("failed to marshal notification body: %v", err)
	}

	msg := &Message{
		Id:        uuid.NewString(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Topic:     topic,
		Message:   string(mByte),
	}

	notificationsChannel <- msg

	return nil
}
