package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"github.com/arjantop/pwned-passwords/client"
	"github.com/arjantop/pwned-passwords/pwnedpasswords"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
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

func main() {
	flag.Parse()

	if *serverAddr == "" || flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	password := flag.Arg(0)

	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: *jaegerEndpoint,
		ServiceName:   "pwned-passwords-client",
	})
	if err != nil {
		log.Fatalf("Could not initialize jaeger exporter: %s", err)
	}
	trace.RegisterExporter(exporter)

	registry := prom.NewRegistry()
	promExporter, err := prometheus.NewExporter(prometheus.Options{
		Registry: registry,
		OnError: func(err error) {
			log.Println(err)
		},
	})
	if err != nil {
		log.Fatalf("Could not initialize prometheus exporter: %s", err)
	}
	view.RegisterExporter(promExporter)

	var once sync.Once
	flush := func() {
		once.Do(func() {
			if exporter != nil {
				log.Println("Flushing stats ...")
				exporter.Flush()
			}

			view.SetReportingPeriod(100 * time.Millisecond)
			log.Println("Flushing prometheus stats ...")
			time.Sleep(time.Second)
			if err := push.FromGatherer("pwned-passwords-client", nil, *promGateway, registry); err != nil {
				log.Printf("Could not flush stats to prometheus: %s", err)
			}
		})
	}

	defer func() {
		flush()
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
