package apiserver

import (
	"context"
	"github.com/golang/glog"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/samsung-cnct/cma-ssh/internal/apiserver"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strconv"

	pb "github.com/samsung-cnct/cma-ssh/pkg/generated/api"
	"github.com/samsung-cnct/cma-ssh/pkg/ui/website"
)

type ServerOptions struct {
	PortNumber int
}

type ApiServer struct {
	Manager manager.Manager
	TcpMux  cmux.CMux
}

type MuxApiServer interface {
	AddServersToMux(options *ServerOptions)
	GetMux() cmux.CMux
}

func NewApiServer(manager manager.Manager, tcpMux cmux.CMux) MuxApiServer {
	return &ApiServer{Manager: manager, TcpMux: tcpMux}
}

func (r *ApiServer) AddServersToMux(options *ServerOptions) {
	r.addGRPCServer(r.TcpMux)
	r.addRestAndWebsite(r.TcpMux, options.PortNumber)
}

func (r *ApiServer) GetMux() cmux.CMux {
	return r.TcpMux
}

func (r *ApiServer) addGRPCServer(tcpMux cmux.CMux) {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterClusterServer(grpcServer, r.newgRPCServiceServer())
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	grpcListener := tcpMux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	// Start servers
	go func() {
		glog.Info("Starting gRPC Server")
		if err := grpcServer.Serve(grpcListener); err != nil {
			glog.Errorf("Unable to start external gRPC server: %q", err)
		}
	}()
}

func (r *ApiServer) addRestAndWebsite(tcpMux cmux.CMux, grpcPortNumber int) {
	httpListener := tcpMux.Match(cmux.HTTP1Fast())

	go func() {
		router := http.NewServeMux()
		website.AddWebsiteHandles(router)
		r.addgRPCRestGateway(router, grpcPortNumber)
		httpServer := http.Server{
			Handler: router,
		}
		glog.Info("Starting HTTP/1 Server")
		err := httpServer.Serve(httpListener)
		if err != nil {
			glog.Errorf("Failed to start http server Serve(): %q", err)
		}
	}()

}

func (r *ApiServer) addgRPCRestGateway(router *http.ServeMux, grpcPortNumber int) {
	dopts := []grpc.DialOption{grpc.WithInsecure()}
	gwmux := runtime.NewServeMux()
	err := pb.RegisterClusterHandlerFromEndpoint(context.Background(), gwmux, "localhost:"+strconv.Itoa(grpcPortNumber), dopts)
	if err != nil {
		glog.Errorf("Failed to register handler from enpoint: %q", err)
	}
	router.Handle("/api/", gwmux)
}

func (r *ApiServer) newgRPCServiceServer() *apiserver.Server {
	return &apiserver.Server{Manager: r.Manager}
}
