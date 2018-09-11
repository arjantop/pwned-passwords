package monitoring

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
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
		var result *multierror.Error
		for _, flushFunc := range flushFuncs {
			if err := flushFunc(); err != nil {
				multierror.Append(result, err)
			}
		}
		return result.ErrorOrNil()
	}
}

func RegisterJaegerExporter(jaegerEndpoint string, serviceName string) (FlushFunc, error) {
	exporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: jaegerEndpoint,
		ServiceName:   serviceName,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "initializing jaeger exporter failed")
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
		return nil, errors.WithMessage(err, "initializing prometheus exporter failed")
	}
	view.RegisterExporter(exporter)

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		return nil, errors.WithMessage(err, "registering grpc views failed")
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

// RegisterJobPrometheusExporter flushes view stats to a prometheus gateway.
// It should be callend only before exiting the program. Flushing has a side-effect of
// modifying the view report period (view.SetReportingPeriod).
func RegisterJobPrometheusExporter(serviceName string, prometheusGateway string) (FlushFunc, error) {
	return registerPrometheusExporterWithFlush(func(registry *prom.Registry) FlushFunc {
		return func() error {
			view.SetReportingPeriod(100 * time.Millisecond)
			time.Sleep(time.Second)
			if err := push.FromGatherer(serviceName, nil, prometheusGateway, registry); err != nil {
				return errors.WithMessage(err, "flushing stats to prometheus failed")
			}
			return nil
		}
	})
}
