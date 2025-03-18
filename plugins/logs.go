package plugins

import (
	"context"
	"errors"
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"runtime"
	"sync"
	"time"
)

var logsChannel chan *Log
var logsChannelOnce sync.Once

// SendLogsFromChannel listens to the logsChannel and sends logs
// to the engine server via gRPC. It logs an error if the connection to the engine server fails,
// if sending a notification fails, or if receiving an acknowledgment fails. It runs indefinitely
// and should be run as a goroutine.
func SendLogsFromChannel() {
	socketDir, err := utils.MkdirJoin(WorkDir, "sockets")
	if err != nil {
		_ = catcher.Error("failed to create socket directory", err, nil)
		os.Exit(1)
	}
	socketFile := utils.FileJoin(socketDir, "engine_server.sock")

	conn, err := grpc.NewClient(fmt.Sprintf("unix://%s", socketFile), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = catcher.Error("failed to connect to engine server", err, map[string]any{})
		os.Exit(1)
	}

	client := NewEngineClient(conn)

	inputClient, err := client.Input(context.Background())
	if err != nil {
		_ = catcher.Error("failed to create input client", err, map[string]any{})
		os.Exit(1)
	}

	logsChannelOnce.Do(func() {
		logsChannel = make(chan *Log, runtime.NumCPU()*1000)
	})

	go func() {
		for {
			log := <-logsChannel

			err := inputClient.Send(log)
			if err != nil {
				_ = catcher.Error("failed to send log", err, map[string]any{})
				os.Exit(1)
			}
		}
	}()

	for {
		_, err := inputClient.Recv()
		if err != nil {
			_ = catcher.Error("failed to receive ack", err, map[string]any{})
			os.Exit(1)
		}
	}
}

// EnqueueLog sends a log to the local logs queue.
// Parameters:
//   - log: The log to enqueue
func EnqueueLog(log *Log) {
	select {
	case logsChannel <- log:
		return
	case <-time.After(1 * time.Second):
		_ = catcher.Error("cannot enqueue log", errors.New("queue is full"), map[string]any{
			"advise": "please consider to increase resources",
			"queue":  "logsChannel",
		})
	}
}
