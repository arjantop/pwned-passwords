package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"encoding/hex"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

var (
	serverAddr   = flag.String("addr", "", "address and port of remote server")
	gcpProjectId = flag.String("gcpProjectId", "", "Google Cloud project id")
)

func main() {
	flag.Parse()

	if *serverAddr == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	prefix := flag.Arg(0)

	var exporter *stackdriver.Exporter
	if *gcpProjectId != "" {
		log.Println("Initializing stackdriver exporter ...")
		var err error
		exporter, err = stackdriver.NewExporter(stackdriver.Options{ProjectID: *gcpProjectId})
		if err != nil {
			log.Fatalf("Could not initialize stackdriver exporter: %s", err)
		}

		view.RegisterExporter(exporter)
		trace.RegisterExporter(exporter)
	}
	defer func() {
		if exporter != nil {
			log.Println("Flushing stats ...")
			exporter.Flush()
		}
	}()

	// For demo purposes
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	conn, err := grpc.Dial(*serverAddr, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not dial: %s", err)
	}
	defer conn.Close()

	client := pwnedpasswords.NewPwnedPasswordsClient(conn)
	r, err := client.ListHashesForPrefix(context.Background(), &pwnedpasswords.ListRequest{
		HashPrefix: prefix,
	})

	if err != nil {
		log.Fatalf("Call failed: %s", err)
	}

	fmt.Println("Hashes:")
	for {
		h, err := r.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Receiving failed: %s", err)
			return
		}
		fmt.Printf(hex.EncodeToString(h.HashSuffix))
		fmt.Println()
	}
}
