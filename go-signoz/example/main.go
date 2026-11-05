// Command example runs the signoz agent against locally-configured endpoints
// and burns CPU in a background goroutine so you can watch the metric climb in
// SigNoz (and the flame graph fill in, if a profiling endpoint is set).
//
// Usage:
//
//	SIGNOZ_METRICS_ENDPOINT=http://localhost:4318 \
//	SIGNOZ_PROFILES_ENDPOINT=http://localhost:4040 \
//	go run ./example
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	signoz "github.com/icelab-research/raidzero/go-signoz"
)

// main starts a demo agent and creates enough work to produce a useful profile.
func main() {
	agent, err := signoz.Start(signoz.Config{
		ServiceName:       "signoz-go-example",
		MetricsEndpoint:   os.Getenv("SIGNOZ_METRICS_ENDPOINT"),
		MetricsHeaders:    map[string]string{"signoz-ingestion-key": os.Getenv("SIGNOZ_INGESTION_KEY")},
		ProfilesEndpoint:  os.Getenv("SIGNOZ_PROFILES_ENDPOINT"),
		ProfilesAuthToken: os.Getenv("SIGNOZ_PROFILES_TOKEN"),
		MetricsInterval:   5 * time.Second,
		ProfileInterval:   5 * time.Second,
		Labels:            map[string]string{"env": "local"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer agent.Stop()
	log.Println("agent started; press Ctrl-C to stop")

	go burnCPU()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Println("shutting down")
}

// burnCPU keeps one core busy so the example produces a non-trivial profile.
func burnCPU() {
	x := 0
	for {
		for i := 0; i < 1e7; i++ {
			x = (x*31 + i) % 1_000_000_007
		}
		_ = x
	}
}
