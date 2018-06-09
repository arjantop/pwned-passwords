package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"

	"net/http"
	_ "net/http/pprof"

	"fmt"

	"github.com/arjantop/pwned-passwords/internal/monitoring"
	"github.com/arjantop/pwned-passwords/internal/storage"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	listenOn       = flag.String("listen", "", "Interface and port the server will listen on")
	dataDir        = flag.String("dataDir", "", "Directory where password data is located")
	jaegerEndpoint = flag.String("jaegerEndpoint", "", "Endpoint of jaeger tracing")
)

const prefixLength = 5

type Server struct {
	storage storage.Storage
}

func (s *Server) ListHashesForPrefix(req *pwnedpasswords.ListRequest, resp pwnedpasswords.PwnedPasswords_ListHashesForPrefixServer) error {
	if len(req.HashPrefix) != prefixLength {
		return status.Errorf(codes.InvalidArgument, "prefix length must be %d", prefixLength)
	}

	hashes, err := s.storage.Get(resp.Context(), req.HashPrefix)
	if err != nil {
		return err
	}

	for _, h := range hashes {
		err := resp.Send(&pwnedpasswords.PasswordHash{
			HashSuffix: h[prefixLength/2:],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

type GrpcServer struct {
	listenOn string
	name     string
	init     func(server *grpc.Server)
	flusher  monitoring.FlushFunc

	started bool
}

func NewGrpcServer(listenOn string, name string, init func(server *grpc.Server)) *GrpcServer {
	return &GrpcServer{
		listenOn: listenOn,
		name:     name,
		init:     init,
	}
}

func (s *GrpcServer) Start() error {
	http.Handle("/debug/", http.StripPrefix("/debug", zpages.Handler))

	if err := s.setUpMonitoring(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", s.listenOn)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Println("Starting server ...")
	srv := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))

	s.init(srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		log.Println("Stopping server ...")
		srv.GracefulStop()
	}()

	s.started = true

	go func() {
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()

	return srv.Serve(lis)
}

func (s *GrpcServer) setUpMonitoring() error {
	var flushers []monitoring.FlushFunc

	flushJaeger, err := monitoring.RegisterJaegerExporter(*jaegerEndpoint, s.name)
	if err != nil {
		return err
	}
	flushers = append(flushers, flushJaeger)

	flushPrometheus, err := monitoring.RegisterPrometheusExporter()
	if err != nil {
		return err
	}
	flushers = append(flushers, flushPrometheus)

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return fmt.Errorf("registering grpc views: %s", err)
	}

	s.flusher = monitoring.CombineFlushFunc(flushers...)

	return nil
}

func (s *GrpcServer) Stop() error {
	if !s.started {
		return nil
	}

	if s.flusher != nil {
		if err := s.flusher(); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if *listenOn == "" || *dataDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	// For demo purposes
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	s := NewGrpcServer(*listenOn, "pwned-passwords", func(srv *grpc.Server) {
		s := &Server{
			storage: &storage.Local{Dir: *dataDir},
		}
		pwnedpasswords.RegisterPwnedPasswordsServer(srv, s)
	})
	defer s.Stop()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
