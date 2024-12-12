package go_sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// notificationsChannel is a channel used to queue notification messages
// that need to be sent to the engine server
var notificationsChannel chan *Message

// Represent the details of a success or failure during the processing of a log. Used as a message body for notifications.
type DataProcessingMessage struct {
	Error      interface{} `json:"error,omitempty"`
	DataType   string      `json:"dataType"`
	DataSource string      `json:"dataSource"`
}

type Topic string

const (
	TOPIC_ENQUEUE_FAILURE          Topic = "enqueue_failure"          // TOPIC_ENQUEUE_FAILURE represents the topic name for enqueue failure notifications.
	TOPIC_ENQUEUE_SUCCESS          Topic = "enqueue_success"          // TOPIC_ENQUEUE_SUCCESS represents the topic name for enqueue success notifications.
	TOPIC_INTEGRATION_FAILURE      Topic = "integration_failure"      // TOPIC_INTEGRATION_FAILURE represents the topic name for integration failure notifications.
	TOPIC_PARSING_FAILURE          Topic = "parsing_failure"          // TOPIC_PARSING_FAILURE represents the topic name for parsing failure notifications.
	TOPIC_ANALYSIS_FAILURE         Topic = "analysis_failure"         // TOPIC_ANALYSIS_FAILURE represents the topic name for analysis failure notifications.
	TOPIC_CORRELATION_FAILURE      Topic = "correlation_failure"      // TOPIC_CORRELATION_FAILURE represents the topic name for correlation failure notifications.
	TOPIC_OUTGOING_REQUEST_FAILURE Topic = "outgoing_request_failure" // TOPIC_OUTGOING_REQUEST_FAILURE represents the topic name for outgoing request failure notifications.
	TOPIC_CEL_EVALATUAION_FAILURE  Topic = "cel_evaluation_failure"   // TOPIC_CEL_EVALUATION_FAILURE represents the topic name for CEL evaluation failure notifications.
)

// SendNotificationsFromChannel listens to the notificationsChannel and sends notifications
// to the engine server via gRPC. It logs errors if the connection to the engine server fails,
// if sending a notification fails, or if receiving an acknowledgment fails. It runs indefinitely
// and should be run as a goroutine.
//
// Returns:
//
//	error: An error object if any error occurs during the process.
func SendNotificationsFromChannel() error {
	conn, err := grpc.NewClient(fmt.Sprintf("unix://%s", path.Join(
		GetCfg().Env.Workdir, "sockets", "engine_server.sock")),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to engine server: %v", err)
	}

	client := NewEngineClient(conn)

	notifyClient, err := client.Notify(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create notify client: %v", err)
	}

	for {
		msg := <-notificationsChannel

		err = notifyClient.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send notification: %v", err)
		}

		_, err := notifyClient.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive notification ack: %v", err)
		}
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
//   - error: Returns an error if the message marshalling fails, otherwise returns nil.
func EnqueueNotification[T any](topic Topic, message T) error {
	mBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal notification body: %v", err)
	}

	msg := &Message{
		Id:        uuid.NewString(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Topic:     string(topic),
		Message:   string(mBytes),
	}

	notificationsChannel <- msg

	return nil
}
