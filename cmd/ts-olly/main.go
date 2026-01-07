package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type config struct {
	port             int
	env              string
	node             string
	logsDir          string
	configDir        string
	parse            bool
	skipFiles        []string
	readExistingLogs bool
}
type application struct {
	config  config
	logger  zerolog.Logger
	wg      sync.WaitGroup
	watcher *fsnotify.Watcher
}

func main() {
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()
	cfg := config{
		skipFiles: []string{
			"searchserver-0.log",
			".gz",
		},
	}

	flag.IntVar(&cfg.port, "port", 2112, "application port")
	flag.StringVar(&cfg.env, "env", "development", "environment (development|staging|production)")
	flag.StringVar(&cfg.node, "node", "", "tableau cluster node id (e.g. node1, node2, etc.)")
	flag.StringVar(&cfg.logsDir, "logsdir", "", "logs directory")
	flag.StringVar(&cfg.configDir, "configdir", "", "config directory")
	flag.BoolVar(&cfg.parse, "parse", false, "parse recognizable logs lines into json")
	flag.BoolVar(&cfg.readExistingLogs, "read-existing-logs", false, "read existing logs")
	flag.Parse()

	logger := zerolog.New(os.Stdout).With().
		Str("node", cfg.node).
		Timestamp().
		Logger()

	path, err := filepath.Abs(cfg.logsDir)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
	cfg.logsDir = path

	app := &application{
		config: cfg,
		logger: logger.With().Logger(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(cancel context.CancelFunc, app *application) {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		app.logger.Info().Str("signal", s.String()).Msg("shutting down")
		cancel()
	}(cancel, app)

	wg := sync.WaitGroup{}
	wg.Add(1)
	runServe(ctx, app, &wg)

	i := 1
WatchForLogsDir:
	for {
		_, err := os.Stat(app.config.logsDir)
		if err != nil && os.IsNotExist(err) {
			app.logger.Warn().Err(err).Msg("error reading logs dir")
			select {
			case <-time.After(time.Duration(i) * time.Second):
				i *= 2
			case <-ctx.Done():
				break WatchForLogsDir
			}
			continue
		}
		if err != nil {
			app.logger.Fatal().Err(err).Msg("unexpected error reading logs dir")
			continue
		}
		break
	}
	if ctx.Err() == nil {
		wg.Add(1)
		runLogs(ctx, app, &wg)
	}
	wg.Wait()
}

func runServe(ctx context.Context, app *application, wg *sync.WaitGroup) {
	go func(ctx context.Context, app *application, wg *sync.WaitGroup) {
		defer wg.Done()
		if err := app.serve(ctx); err != nil {
			app.logger.Fatal().Err(err)
		}
	}(ctx, app, wg)
}

func runLogs(ctx context.Context, app *application, wg *sync.WaitGroup) {
	go func(ctx context.Context, app *application, wg *sync.WaitGroup) {
		defer wg.Done()
		lines := app.logs(ctx)
		for l := range lines {
			outputLine(app.logger, l)
		}
	}(ctx, app, wg)
}

func outputLine(logger zerolog.Logger, l line) {
	lineLogger := logger.With().
		Str("filename", l.filename).
		Stringer("fileid", l.fileId).
		Str("process", l.processName).
		Uint8("processid", l.processId).
		Int("line", l.Num).
		Int64("offset", l.SeekInfo.Offset).
		Logger()
	if l.Err != nil {
		lineLogger.Err(l.Err).Send()
	} else {
		var level zerolog.Level
		var log *zerolog.Event
		var err error
		if l.Level() != "" {
			level, err = zerolog.ParseLevel(l.Level())
			if err != nil {
				lineLogger.Err(err).Str("component", "ts-olly").Send()
				log = lineLogger.Log().Str("level", l.Level())
			} else {
				log = lineLogger.WithLevel(level).Str("component", l.component)
			}
		} else {
			log = lineLogger.Log().Str("component", l.component)
		}
		if lineBytes := []byte(l.Line.Text); json.Valid(lineBytes) {
			log.RawJSON("message", lineBytes).Send()
		} else {
			log.Str("message", l.Line.Text).Send()
		}
	}
}
