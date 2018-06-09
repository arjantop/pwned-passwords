package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"fmt"

	"github.com/arjantop/pwned-passwords/client"
	"github.com/arjantop/pwned-passwords/internal/monitoring"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

var (
	serverAddr     = flag.String("addr", "", "address and port of remote server")
	promGateway    = flag.String("promGateway", "", "URL of Prometheus push gateway")
	jaegerEndpoint = flag.String("jaegerEndpoint", "", "Endpoint of jaeger tracing")
)

func setUpMonitoring(serviceName string) (monitoring.FlushFunc, error) {
	flushJaeger, err := monitoring.RegisterJaegerExporter(*jaegerEndpoint, serviceName)
	if err != nil {
		return nil, err
	}

	flushPrometheus, err := monitoring.RegisterJobPrometheusExporter(serviceName, *promGateway)
	if err != nil {
		return nil, err
	}

	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		return nil, fmt.Errorf("registering grpc views: %s", err)
	}

	return monitoring.CombineFlushFunc(flushJaeger, flushPrometheus), nil
}

func main() {
	flag.Parse()

	if *serverAddr == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	password := flag.Arg(0)

	flush, err := setUpMonitoring("pwned-passwords-client")
	if err != nil {
		log.Fatalf("Failed to set up monitoring: %s", err)
	}

	defer func() {
		if err := flush(); err != nil {
			log.Printf("Could not flush: %s", err)
		}
	}()

	// Register the view to collect gRPC client stats.
	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		log.Fatalf("Could not register views: %s", err)
	}

	// For demo purposes
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	conn, err := grpc.Dial(*serverAddr, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not dial: %s", err)
	}
	defer conn.Close()

	c := client.Client{
		C: pwnedpasswords.NewPwnedPasswordsClient(conn),
	}

	pwned, err := c.IsPasswordPwned(context.Background(), password)
	if err != nil {
		log.Printf("Pwned password call failed: %s", err)
		return
	}

	time.Sleep(1 * time.Second)

	if pwned {
		log.Println("The password has been pwned")
		flush()
		os.Exit(1)
	} else {
		log.Println("The password has not been pwned yet")
	}
}
