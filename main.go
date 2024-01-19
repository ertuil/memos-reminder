package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

var (
	config_file   string
	database_file string
	debug         bool
)

func InitOptions() {
	flag.StringVar(&config_file, "config", "config.yaml", "Config file")
	flag.StringVar(&database_file, "database", "data.db", "Database file")
	flag.BoolVar(&debug, "verbose", false, "Debug mode")

	flag.Parse()
}

func InitLogger() {
	opts := slog.HandlerOptions{}
	if debug {
		opts.AddSource = true
		opts.Level = slog.LevelDebug
	} else {
		opts.AddSource = false
		opts.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func main() {
	InitOptions()
	InitLogger()

	err := LoadConfig(config_file)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	InitDatabase()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go TimerServe(ctx)
	go HTTPServe(ctx)

	s := <-c
	slog.Info("Got signal", "signal", s)
	cancel()
	time.Sleep(time.Second)
}
