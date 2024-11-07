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
type DataProcessingMessage struct {
	Cause      *string `json:"cause,omitempty"`
	DataType   string  `json:"dataType"`
	DataSource string  `json:"dataSource"`
}

// Represents a notification message to be sent to the backend in the event of a failure in an integration.
type IntegrationFailureMessage struct {
	Cause           string  `json:"cause"`
	IntegrationName string  `json:"integrationName"`
	Tenant          *string `json:"tenant,omitempty"`
}

const (
	TOPIC_ENQUEUE_FAILURE     = "enqueue_failure"     // TOPIC_ENQUEUE_FAILURE represents the topic name for enqueue failure notifications.
	TOPIC_ENQUEUE_SUCCESS     = "enqueue_success"     // TOPIC_ENQUEUE_SUCCESS represents the topic name for enqueue success notifications.
	TOPIC_INTEGRATION_FAILURE = "integration_failure" // TOPIC_INTEGRATION_FAILURE represents the topic name for integration failure notifications.
	TOPIC_PARSING_FAILURE     = "parsing_failure"     // TOPIC_PARSING_FAILURE represents the topic name for parsing failure notifications.
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
//   - message: The notification message to be sent. Must be a JSON serializable object.
//
// Returns:
//   - *logger.Error: Returns an error if the message marshalling fails, otherwise returns nil.
func EnqueueNotification[T any](topic string, message T) *logger.Error {
	mBytes, err := json.Marshal(message)
	if err != nil {
		return Logger().ErrorF("failed to marshal notification body: %v", err)
	}

	msg := &Message{
		Id:        uuid.NewString(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Topic:     topic,
		Message:   string(mBytes),
	}

	notificationsChannel <- msg

	return nil
}
