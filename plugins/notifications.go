package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/protobuf/types/known/emptypb"

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
	DataType   string `json:"dataType"`
	DataSource string `json:"dataSource"`
}

type Topic string

const (
	TopicEnqueueSuccess     Topic = "enqueue_success"     // represents the topic name for enqueue success notifications.
	TopicParsingDropped     Topic = "parsing_dropped"     // represents the topic name for parsing dropped events.
	TopicAnalysisDropped    Topic = "analysis_dropped"    // represents the topic name for analysis dropped events.
	TopicCorrelationDropped Topic = "correlation_dropped" // represents the topic name for correlation dropped events.
)

// SendNotificationsFromChannel listens to the notificationsChannel and sends notifications
// to the engine server via gRPC. It logs an error if the connection to the engine server fails,
// if sending a notification fails, or if receiving an acknowledgment fails. It runs indefinitely
// and should be run as a goroutine.
func SendNotificationsFromChannel(pluginName string) {
	processName := fmt.Sprintf("plugin_%s", pluginName)

	socketDir, err := utils.MkdirJoin(WorkDir, "sockets")
	if err != nil {
		_ = catcher.Error("failed to create socket directory", err, map[string]any{
			"process": processName,
		})
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	socketFile := socketDir.FileJoin("engine_server.sock")

	conn, err := grpc.NewClient(fmt.Sprintf("unix://%s", socketFile),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = catcher.Error("failed to connect to engine server", err, map[string]any{
			"socket":  socketFile,
			"process": processName,
		})
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	client := NewEngineClient(conn)

	notifyClient, err := client.Notify(context.Background())
	if err != nil {
		_ = catcher.Error("failed to create notify client", err, map[string]any{
			"socket":  socketFile,
			"process": processName,
		})
		time.Sleep(5 * time.Second)
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
				_ = catcher.Error("failed to send notification", err, map[string]any{
					"process": processName,
				})
				time.Sleep(5 * time.Second)
				os.Exit(1)
			}
		}
	}()

	for {
		_, err := notifyClient.Recv()
		if err != nil {
			_ = catcher.Error("failed to receive notification ack", err, map[string]any{
				"process": processName,
			})
			time.Sleep(5 * time.Second)
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
func EnqueueNotification[T any](topic Topic, message T, pluginName string) error {
	processName := fmt.Sprintf("plugin_%s", pluginName)

	mBytes, err := json.Marshal(message)
	if err != nil {
		return catcher.Error("failed to marshal notification body", err, map[string]any{
			"process": processName,
		})
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
			"advise":  "please consider to increase resources",
			"process": processName,
		})
	}
}

type notificationServer struct {
	UnimplementedNotificationServer
	notificationFunction func(context.Context, *Message) (*emptypb.Empty, error)
}

func (p *notificationServer) Notify(ctx context.Context, message *Message) (*emptypb.Empty, error) {
	return p.notificationFunction(ctx, message)
}

func InitNotificationPlugin(name string, notificationFunction func(ctx context.Context, message *Message) (*emptypb.Empty, error)) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	processName := fmt.Sprintf("plugin_%s", name)

	// Create sockets folder
	socketsFolder, err := utils.MkdirJoin(WorkDir, "sockets")
	if err != nil {
		return catcher.Error("cannot create sockets folder", err, map[string]any{
			"process": processName,
		})
	}

	socketFile := socketsFolder.FileJoin(fmt.Sprintf("%s_notification.sock", name))

	// Clean up any existing socket file
	err = os.Remove(socketFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return catcher.Error("cannot remove socket", err, map[string]any{
				"socket":  socketFile,
				"process": processName,
			})
		}
	}

	// Set up a deferred cleanup function to ensure the socket is removed on exit
	defer func() {
		err := os.Remove(socketFile)
		if err != nil {
			if !os.IsNotExist(err) {
				_ = catcher.Error("cannot remove socket", err, map[string]any{
					"socket":  socketFile,
					"process": processName,
				})
			}
		}
	}()

	unixAddress, err := net.ResolveUnixAddr("unix", socketFile)
	if err != nil {
		return catcher.Error("cannot resolve unix socket", err, map[string]any{
			"socket":  socketFile,
			"process": processName,
		})
	}

	listener, err := net.ListenUnix("unix", unixAddress)
	if err != nil {
		return catcher.Error("cannot listen to unix socket", err, map[string]any{
			"socket":      socketFile,
			"unixAddress": unixAddress.String(),
			"process":     processName,
		})
	}

	defer func(listener *net.UnixListener) {
		_ = listener.Close()
	}(listener)

	// Create a gRPC server
	grpcServer := grpc.NewServer()

	newNotificationServer := &notificationServer{
		notificationFunction: notificationFunction,
	}

	RegisterNotificationServer(grpcServer, newNotificationServer)

	// Start the server in a goroutine so we can handle shutdown signals
	serverErrors := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			_ = catcher.Error("cannot serve grpc", err, map[string]any{
				"socket":      socketFile,
				"unixAddress": unixAddress.String(),
				"process":     processName,
			})
			serverErrors <- err
		}
	}()

	// Wait for a shutdown signal or server error
	select {
	case <-sigChan:
		catcher.Info("shutdown signal received, stopping server", map[string]any{
			"socket":  socketFile,
			"process": processName,
		})
	case err := <-serverErrors:
		return catcher.Error("server error, shutting down", err, map[string]any{
			"socket":  socketFile,
			"process": processName,
		})
	}

	// Graceful shutdown
	grpcServer.GracefulStop()

	// Give time for connections to close
	time.Sleep(time.Second)

	return nil
}
