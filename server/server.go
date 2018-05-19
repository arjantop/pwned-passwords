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

	"github.com/arjantop/pwned-passwords/internal/filename"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	listenOn = flag.String("listen", "", "Interface and port the server will listen on")
	dataDir  = flag.String("dataDir", "", "Directory where password data is located")
)

type Storage interface {
	Get(ctx context.Context, key string) ([][]byte, error)
}

type LocalStorage struct {
	dir string
}

func (s *LocalStorage) Get(ctx context.Context, key string) ([][]byte, error) {
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
			HashSuffix: h,
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

	lis, err := net.Listen("tcp", *listenOn)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := &Server{
		storage: &LocalStorage{dir: *dataDir},
	}

	log.Println("Starting server ...")
	srv := grpc.NewServer()
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
