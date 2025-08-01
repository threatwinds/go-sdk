package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// notificationsChannel is a channel used to queue notification messages
// that need to be sent to the engine server
var notificationsChannel chan *Message
var notificationsChannelOnce sync.Once

// DataProcessingMessage represent the details of a success or failure during the processing of a log. Used as a message body for notifications.
type DataProcessingMessage struct {
	Error      *catcher.SdkError `json:"error,omitempty"`
	DataType   string            `json:"dataType"`
	DataSource string            `json:"dataSource"`
}

type Topic string

const (
	TopicEnqueueSuccess     Topic = "enqueue_success"     // represents the topic name for enqueue success notifications.
	TopicIntegrationFailure Topic = "integration_failure" // represents the topic name for integration failure notifications.
	TopicParsingFailure     Topic = "parsing_failure"     // represents the topic name for parsing failure notifications.
	TopicAnalysisFailure    Topic = "analysis_failure"    // represents the topic name for analysis failure notifications.
	TopicCorrelationFailure Topic = "correlation_failure" // represents the topic name for correlation failure notifications.
)

// SendNotificationsFromChannel listens to the notificationsChannel and sends notifications
// to the engine server via gRPC. It logs an error if the connection to the engine server fails,
// if sending a notification fails, or if receiving an acknowledgment fails. It runs indefinitely
// and should be run as a goroutine.
func SendNotificationsFromChannel() {
	socketDir, err := utils.MkdirJoin(WorkDir, "sockets")
	if err != nil {
		_ = catcher.Error("failed to create socket directory", err, nil)
		os.Exit(1)
	}
	socketFile := socketDir.FileJoin("engine_server.sock")

	conn, err := grpc.NewClient(fmt.Sprintf("unix://%s", socketFile),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = catcher.Error("failed to connect to engine server", err, nil)
		os.Exit(1)
	}

	client := NewEngineClient(conn)

	notifyClient, err := client.Notify(context.Background())
	if err != nil {
		_ = catcher.Error("failed to create notify client", err, nil)
		os.Exit(1)
	}

	notificationsChannelOnce.Do(func() {
		notificationsChannel = make(chan *Message, runtime.NumCPU()*100)
	})

	go func() {
		for {
			msg := <-notificationsChannel

			err = notifyClient.Send(msg)
			if err != nil {
				_ = catcher.Error("failed to send notification", err, nil)
				os.Exit(1)
			}
		}
	}()

	for {
		_, err := notifyClient.Recv()
		if err != nil {
			_ = catcher.Error("failed to receive notification ack", err, nil)
			os.Exit(1)
		}
	}
}

// EnqueueNotification sends a notification message to a specified topic.
// It marshals the NotificationMessage into JSON format and sends it to the notification channel.
//
// Parameters:
//   - topic: The topic to which the notification message will be sent.
//   - message: The notification message to be sent. Must be a JSON serializable object.
//
// Returns:
//   - error: Returns an error if the message marshaling fails, otherwise returns nil.
func EnqueueNotification[T any](topic Topic, message T) error {
	mBytes, err := json.Marshal(message)
	if err != nil {
		return catcher.Error("failed to marshal notification body", err, nil)
	}

	msg := &Message{
		Id:        uuid.NewString(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Topic:     string(topic),
		Message:   string(mBytes),
	}

	select {
	case notificationsChannel <- msg:
		return nil
	case <-time.After(1 * time.Second):
		return catcher.Error("cannot enqueue message", errors.New("queue is full"), map[string]any{
			"advise": "please consider to increase resources",
			"queue":  "notificationsChannel",
		})
	}
}
