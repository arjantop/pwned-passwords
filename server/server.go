package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/arjantop/pwned-passwords/internal/grpcbase"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/arjantop/pwned-passwords/internal/storage"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"go.opencensus.io/trace"
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

type server struct {
	storage storage.Storage
}

func (s *server) ListHashesForPrefix(req *pwnedpasswords.ListRequest, resp pwnedpasswords.PwnedPasswords_ListHashesForPrefixServer) error {
	if len(req.HashPrefix) != prefixLength {
		return status.Errorf(codes.InvalidArgument, "prefix length must be %d", prefixLength)
	}

	hashes, err := s.storage.Get(resp.Context(), req.HashPrefix)
	if err != nil {
		log.Printf("Faled fething from storage for prefix '%s': %v", req.HashPrefix, err)
		return status.Error(codes.Internal, "Something went wrong")
	}

	for _, h := range hashes {
		err := resp.Send(&pwnedpasswords.PasswordHash{
			Hash: h,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func registerHttpServer(conn *grpc.ClientConn) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux()
	if err := pwnedpasswords.RegisterPwnedPasswordsHandler(ctx, mux, conn); err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := http.ListenAndServe(":8990", mux); err != nil {
			log.Fatal(err)
		}
	}()
}

func main() {
	flag.Parse()

	if *listenOn == "" || *dataDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	// For demo purposes
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	s := grpcbase.NewServer(*listenOn, "pwned-passwords", *jaegerEndpoint, func(srv *grpc.Server) {
		s := &server{
			storage: storage.NewLocalStorage(*dataDir),
		}
		pwnedpasswords.RegisterPwnedPasswordsServer(srv, s)
	})
	defer s.Stop()

	if err := s.StartWithClient(registerHttpServer); err != nil {
		log.Fatal(err)
	}
}
