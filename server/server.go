package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path"

	"net/http"
	_ "net/http/pprof"

	"github.com/arjantop/pwned-passwords/internal/filename"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
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

type Storage interface {
	Get(ctx context.Context, key string) ([][]byte, error)
}

type LocalStorage struct {
	dir string
}

func (s *LocalStorage) Get(ctx context.Context, key string) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "LocalStorage.Get")
	defer span.End()

	filePath := filename.PathFor(key, ".bin")

	buf, err := ioutil.ReadFile(path.Join(s.dir, filePath))
	if err != nil {
		return nil, err
	}

	numHashes := len(buf) / 20
	hashes := make([][]byte, 0, numHashes)

	for i := 0; i < numHashes; i++ {
		hashes = append(hashes, buf[i*20:(i+1)*20])
	}

	return hashes, err
}

const prefixLength = 5

type Server struct {
	storage Storage
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

func main() {
	flag.Parse()

	if *listenOn == "" || *dataDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	go func() {
		http.Handle("/debug/", http.StripPrefix("/debug", zpages.Handler))
		log.Fatal(http.ListenAndServe(":6060", nil))
	}()

	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: *jaegerEndpoint,
		ServiceName:   "pwned-passwords",
	})
	if err != nil {
		log.Fatalf("Could not initialize jaeger exporter: %s", err)
	}
	trace.RegisterExporter(exporter)

	pExporter, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		log.Fatalf("Could not initialize prometheus exporter: %s", err)
	}
	view.RegisterExporter(pExporter)

	go func() {
		http.Handle("/metrics", pExporter)
		log.Fatal(http.ListenAndServe(":9999", nil))
	}()

	// For demo purposes
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Fatalf("Could not register views: %s", err)
	}

	lis, err := net.Listen("tcp", *listenOn)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := &Server{
		storage: &LocalStorage{dir: *dataDir},
	}

	log.Println("Starting server ...")
	srv := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	pwnedpasswords.RegisterPwnedPasswordsServer(srv, s)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		log.Println("Stopping server ...")
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
