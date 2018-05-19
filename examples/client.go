package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("addr", "", "address and port of remote server")
)

func main() {
	flag.Parse()

	if *serverAddr == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	prefix := flag.Arg(0)

	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not dial: %s", err)
	}

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
			log.Fatalf("Receiving failed: %s", err)
		}
		fmt.Printf(hex.EncodeToString(h.HashSuffix))
		fmt.Println()
	}
}
