package grpcutils

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/arjantop/pwned-passwords/internal/monitoring"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
)

type Server struct {
	listenOn       string
	name           string
	jaegerEndpoint string
	init           func(server *grpc.Server)
	flusher        monitoring.FlushFunc

	started bool
}

func NewServer(listenOn string, name string, jaegerEndpoint string, init func(server *grpc.Server)) *Server {
	return &Server{
		listenOn:       listenOn,
		name:           name,
		jaegerEndpoint: jaegerEndpoint,
		init:           init,
	}
}

func (s *Server) Start() error {
	http.Handle("/debug/", http.StripPrefix("/debug", zpages.Handler))

	if err := s.setUpMonitoring(); err != nil {
		return err
	}

	lis, err := net.Listen("tcp", s.listenOn)
	if err != nil {
		return errors.WithMessage(err, "failed to listen")
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

func (s *Server) setUpMonitoring() error {
	var flushers []monitoring.FlushFunc

	if s.jaegerEndpoint != "" {
		flushJaeger, err := monitoring.RegisterJaegerExporter(s.jaegerEndpoint, s.name)
		if err != nil {
			return err
		}
		flushers = append(flushers, flushJaeger)
	}

	flushPrometheus, err := monitoring.RegisterPrometheusExporter()
	if err != nil {
		return err
	}
	flushers = append(flushers, flushPrometheus)

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return errors.WithMessage(err, "registering grpc views failed")
	}

	s.flusher = monitoring.CombineFlushFunc(flushers...)

	return nil
}

func (s *Server) Stop() error {
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
