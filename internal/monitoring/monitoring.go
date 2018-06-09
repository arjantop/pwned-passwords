package monitoring

import (
	"fmt"
	"net/http"

	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

type FlushFunc func() error

func CombineFlushFunc(flushFuncs ...FlushFunc) FlushFunc {
	return func() error {
		for _, flushFunc := range flushFuncs {
			flushFunc()
		}
	}
}

func RegisterJaegerExporter(jaegerEndpoint string, serviceName string) (FlushFunc, error) {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: jaegerEndpoint,
		ServiceName:   serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing jaeger exporter: %s", err)
	}
	trace.RegisterExporter(exporter)

	return func() error {
		exporter.Flush()
		return nil
	}, nil
}

func registerPrometheusExporterWithFlush(flush func(registry *prom.Registry) FlushFunc) (FlushFunc, error) {
	registry := prom.NewRegistry()
	exporter, err := prometheus.NewExporter(prometheus.Options{
		Registry: registry,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing prometheus exporter: %s", err)
	}
	view.RegisterExporter(exporter)

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return nil, fmt.Errorf("registering grpc views: %s", err)
	}

	http.Handle("/metrics", exporter)

	return flush(registry), nil
}

func RegisterPrometheusExporter() (FlushFunc, error) {
	return registerPrometheusExporterWithFlush(func(registry *prom.Registry) FlushFunc {
		return func() error {
			return nil
		}
	})
}

//RegisterJobPrometheusExporter flushes view stats to a prometheus gateway.
//It should be callend only before exiting the program. Flushing has a side-effect of
//modifying the view report period (view.SetReportingPeriod).
func RegisterJobPrometheusExporter(serviceName string, prometheusGateway string) (FlushFunc, error) {
	return registerPrometheusExporterWithFlush(func(registry *prom.Registry) FlushFunc {
		return func() error {
			view.SetReportingPeriod(100 * time.Millisecond)
			time.Sleep(time.Second)
			if err := push.FromGatherer(serviceName, nil, prometheusGateway, registry); err != nil {
				return fmt.Errorf("flushing stats to prometheus: %s", err)
			}
			return nil
		}
	})
}
