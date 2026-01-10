package plugins

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/threatwinds/go-sdk/catcher"
	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/grpc"
)

type analysisServer struct {
	UnimplementedAnalysisServer
	analysisFunction func(*Event, grpc.ServerStreamingServer[Alert]) error
}

func (p *analysisServer) Analyze(event *Event, srv grpc.ServerStreamingServer[Alert]) error {
	return p.analysisFunction(event, srv)
}

// InitAnalysisPlugin initializes a gRPC-based analysis plugin with a specified name and analysis function.
// It sets up a UNIX socket, handles lifecycle events, and manages graceful shutdowns upon termination signals.
// Locks until shutdown is complete or an error occurs.
func InitAnalysisPlugin(name string, analysisFunction func(*Event, grpc.ServerStreamingServer[Alert]) error) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create sockets folder
	socketsFolder, err := utils.MkdirJoin(WorkDir, "sockets")
	if err != nil {
		return catcher.Error("cannot create sockets folder", err, nil)
	}

	socket := socketsFolder.FileJoin(fmt.Sprintf("%s_analysis.sock", name))

	// Clean up any existing socket file
	err = os.Remove(socket)
	if err != nil {
		if !os.IsNotExist(err) {
			return catcher.Error("cannot remove socket", err, nil)
		}
	}

	// Set up a deferred cleanup function to ensure the socket is removed on exit
	defer func() {
		err := os.Remove(socket)
		if err != nil {
			if !os.IsNotExist(err) {
				_ = catcher.Error("cannot remove socket", err, nil)
			}
		}
	}()

	unixAddress, err := net.ResolveUnixAddr("unix", socket)
	if err != nil {
		return catcher.Error("cannot resolve unix socket", err, map[string]any{})
	}

	listener, err := net.ListenUnix("unix", unixAddress)
	if err != nil {
		return catcher.Error("cannot listen to unix socket", err, map[string]any{})
	}

	defer func(listener *net.UnixListener) {
		_ = listener.Close()
	}(listener)

	// Create a gRPC server
	grpcServer := grpc.NewServer()

	newAnalysisServer := &analysisServer{
		analysisFunction: analysisFunction,
	}

	RegisterAnalysisServer(grpcServer, newAnalysisServer)

	// Start the server in a goroutine so we can handle shutdown signals
	serverErrors := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			_ = catcher.Error("cannot serve grpc", err, map[string]any{})
			serverErrors <- err
		}
	}()

	// Wait for a shutdown signal or server error
	select {
	case <-sigChan:
		catcher.Info("shutdown signal received, stopping server", nil)
	case err := <-serverErrors:
		return catcher.Error("server error, shutting down", err, nil)
	}

	// Graceful shutdown
	grpcServer.GracefulStop()

	// Give time for connections to close
	time.Sleep(time.Second)

	return nil
}
