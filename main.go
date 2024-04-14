package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d2r2/go-logger"
	"github.com/jessevdk/go-flags"
	"github.com/mfojtik/go-sofar-prometheus/pkg/scraper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	generationNowGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "sofar",
		Name:      "generation_now",
	})
	generationTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "sofar",
		Name:      "generation_total",
	})
	generationTodayGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "sofar",
		Name:      "generation_today",
	})
)

var opts struct {
	Verbose []bool `short:"v" long:"verbose" description:"Show verbose debug information"`

	SofarAddr   string `short:"i" long:"sofar-addr" description:"sofar invertor address" default:"solar:8899"`
	SofarSerial uint   `short:"s" long:"sofar-serial" description:"sofar invertor serial" default:""`
	ListenAddr  string `short:"l" long:"listen-addr" description:"listen address:port" required:"true" default:":2112"`

	ReadSeconds time.Duration `long:"interval" description:"interval between measurements" default:"15s"`
}

var log = logger.NewPackageLogger("dht",
	//logger.DebugLevel,
	logger.InfoLevel,
)

func recordMetrics() {
	for {
		s := scraper.New(opts.SofarAddr, int64(opts.SofarSerial))
		result, err := s.Scrape()
		if err != nil {
			log.Infof("error scraping: %v", err)
			time.Sleep(opts.ReadSeconds)
			continue
		}

		generationNowGauge.Set(float64(result.GenerationNow))
		generationTodayGauge.Set(float64(result.GenerationToday))
		generationTotalGauge.Set(float64(result.GenerationTotal))

		time.Sleep(opts.ReadSeconds)
	}
}

func main() {
	defer logger.FinalizeLogger()
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	if opts.SofarSerial == 0 {
		log.Fatal("sofar serial number must be specified")
	}
	logger.ChangePackageLogLevel("sofar", logger.InfoLevel)

	server := &http.Server{
		Addr: opts.ListenAddr,
	}

	go recordMetrics()
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		log.Infof("Starting HTTP server on %s ...", opts.ListenAddr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Infof("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}
