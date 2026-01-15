package plugins

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type correlationServer struct {
	UnimplementedCorrelationServer
	correlationFunction func(context.Context, *Alert) (*emptypb.Empty, error)
}

func (p *correlationServer) Correlate(ctx context.Context, alert *Alert) (*emptypb.Empty, error) {
	return p.correlationFunction(ctx, alert)
}

// InitCorrelationPlugin initializes a correlation plugin with a given name and correlation function for gRPC communication.
// It sets up a Unix socket, creates a gRPC server, and handles lifecycle management, including shutdown and cleanup.
// Locks until shutdown is complete or an error occurs.
func InitCorrelationPlugin(name string, correlationFunction func(ctx context.Context, alert *Alert) (*emptypb.Empty, error)) error {
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

	socketFile := socketsFolder.FileJoin(fmt.Sprintf("%s_correlation.sock", name))

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

	newCorrelationServer := &correlationServer{
		correlationFunction: correlationFunction,
	}

	RegisterCorrelationServer(grpcServer, newCorrelationServer)

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
			"socket":      socketFile,
			"unixAddress": unixAddress.String(),
			"process":     processName,
		})
	case err := <-serverErrors:
		return catcher.Error("server error, shutting down", err, map[string]any{
			"socket":      socketFile,
			"unixAddress": unixAddress.String(),
			"process":     processName,
		})
	}

	// Graceful shutdown
	grpcServer.GracefulStop()

	// Give time for connections to close
	time.Sleep(time.Second)

	return nil
}
