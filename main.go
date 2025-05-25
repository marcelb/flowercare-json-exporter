package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/marcelb/flowercare-json-exporter/internal/collector"
	"github.com/marcelb/flowercare-json-exporter/internal/config"
	"github.com/marcelb/flowercare-json-exporter/internal/updater"
	"github.com/sirupsen/logrus"
)

var (
	log = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableTimestamp: true,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
)

func main() {
	config, err := config.Parse(log)
	if err != nil {
		log.Fatalf("Error in configuration: %s", err)
	}

	log.SetLevel(logrus.Level(config.LogLevel))
	log.Infof("Bluetooth Device: %s", config.Device)

	provider, err := updater.New(log, config.Device, config.RefreshTimeout, config.Retry)
	if err != nil {
		log.Fatalf("Error creating device: %s", err)
	}

	for _, s := range config.Sensors {
		log.Infof("Sensor: %s", s)
		provider.AddSensor(s)
	}

	c := &collector.Flowercare{
		Log:           log,
		Source:        provider.GetData,
		Sensors:       config.Sensors,
		StaleDuration: config.StaleDuration,
	}

	http.HandleFunc("/sensors", sensorsJSONHandler(c, log))
	http.Handle("/", http.RedirectHandler("/sensors", http.StatusFound))

	go func() {
		log.Infof("Listen on %s...", config.ListenAddr)
		log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
	}()

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	startSignalHandler(ctx, wg, cancel)
	startScheduleLoop(ctx, wg, config, provider)
	provider.Start(ctx, wg)

	log.Info("Exporter is started.")
	wg.Wait()
	log.Info("Shutdown complete.")
}

func sensorsJSONHandler(c *collector.Flowercare, log *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := c.CollectDataAsStructs()
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Errorf("Failed to marshal data to JSON: %s", err)
			http.Error(w, "Failed to marshal data to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if _, err := w.Write(jsonData); err != nil {
			log.Errorf("Failed to write JSON response: %s", err)
		}
	}
}

func startSignalHandler(ctx context.Context, wg *sync.WaitGroup, cancel func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		log.Debug("Signal handler ready.")
		<-sigCh
		log.Debug("Got shutdown signal.")
		signal.Reset()
		cancel()
	}()
}

func startScheduleLoop(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, provider *updater.Updater) {
	wg.Add(1)

	refresher := time.NewTicker(cfg.RefreshDuration)
	provider.UpdateAll(time.Now())

	go func() {
		defer wg.Done()

		log.Debug("Schedule loop ready.")
		for {
			select {
			case <-ctx.Done():
				log.Debug("Shutting down refresh loop")
				return
			case now := <-refresher.C:
				log.Debugf("Updating all at %s", now)
				provider.UpdateAll(now)
			}
		}
	}()
}
