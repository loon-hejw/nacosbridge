package service

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Metrics struct {
	Addr string
}

func NewMetrics(addr string) *Metrics {
	return &Metrics{Addr: addr}
}

func (c *Metrics) Start(ctx context.Context) error {
	l := log.Log.WithName("metrics")

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	}))

	handler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	handler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	handler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	handler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	handler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	handler.Handle("/debug/pprof/block", pprof.Handler("block"))
	handler.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	handler.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	handler.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	srv := &http.Server{Addr: c.Addr, Handler: handler}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			l.Error(err, "shutdown metrics server")
		}
	}()

	return srv.ListenAndServe()
}
