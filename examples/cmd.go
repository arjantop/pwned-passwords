package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/arjantop/pwned-passwords/client"
	"github.com/arjantop/pwned-passwords/internal/monitoring"
	"github.com/arjantop/pwned-passwords/internal/tracing"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	"github.com/pkg/errors"
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
	var flushers []monitoring.FlushFunc

	if *jaegerEndpoint != "" {
		flushJaeger, err := monitoring.RegisterJaegerExporter(*jaegerEndpoint, serviceName)
		if err != nil {
			return nil, err
		}

		flushers = append(flushers, flushJaeger)
	}

	flushPrometheus, err := monitoring.RegisterJobPrometheusExporter(serviceName, *promGateway)
	if err != nil {
		return nil, err
	}

	flushers = append(flushers, flushPrometheus)

	if err := view.Register(ocgrpc.DefaultClientViews...); err != nil {
		return nil, errors.WithMessage(err, "registering grpc views faield")
	}

	return monitoring.CombineFlushFunc(flushers...), nil
}

func main() {
	flag.Parse()

	if *serverAddr == "" || flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	passwords := flag.Args()

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

	ctx, span := trace.StartSpan(context.Background(), "Cmd")
	defer span.End()

	for _, password := range passwords {
		pwned, err := c.IsPasswordPwned(ctx, password)
		if err != nil {
			log.Printf("Pwned password call failed: %s", err)
			tracing.RecordError(span, err)
			return
		}

		if pwned {
			log.Println("The password has been pwned")
		} else {
			log.Println("The password has not been pwned yet")
		}
	}

}
