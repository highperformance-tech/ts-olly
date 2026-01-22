package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/fsnotify/fsnotify"
	"github.com/highperformance-tech/ts-olly/cmd/ts-olly/process"
	"github.com/highperformance-tech/ts-olly/internal/pipeline"
	"github.com/nxadm/tail"
)

type event struct {
	fsnotify.Event
	fileId fileId
}

func (f event) Empty() bool {
	return f.fileId == 0 && f.Event.Op == 0
}

type tailedFile struct {
	*tail.Tail
	fileId      fileId
	processName string
	processId   uint8
	component   string
	logFormat   string
}

type line struct {
	*tail.Line
	filename    string
	fileId      fileId
	processName string
	processId   uint8
	component   string
}

func (l line) String() string {
	return fmt.Sprintf("File: %s\nFile ID: %s\nSeekInfo: {Offset: %d, Whence: %d}\nTime: %s\nLine: %d\nText: %s\nError: %s\n", l.filename, l.fileId, l.SeekInfo.Offset, l.SeekInfo.Whence, l.Time, l.Num, l.Text, l.Err)
}

func (l line) Level() string {
	return getLevel(l.Text)
}

type fileId uint64

func (f fileId) String() string {
	return strconv.FormatUint(uint64(f), 16)
}

func (app *application) logs(ctx context.Context) <-chan line {
	app.logger.Info().Str("component", "logprocessor").Msg("starting")
	defer app.logger.Info().Str("component", "logprocessor").Msg("started")

	// Create a seekInfo cache to store the initial size/offset of the file
	// In order to gracefully handle file renames, we don't key by path but instead by fileId.
	seekInfoCache := &sync.Map{}

	// Create a map for storing/getting whether we are tailing a given fileId
	tailing := &sync.Map{}

	// Create a map for storing pending files awaiting config directory creation
	pendingFiles := &sync.Map{}

	// Initialize watcher for logs directory
	w, err := fsnotify.NewWatcher()
	if err != nil {
		app.logger.Fatal().Err(err)
	}
	app.watcher = w

	// Initialize watcher for config directory to detect new process instances
	configWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		app.logger.Fatal().Err(err)
	}

	// Channel for retrying pending files when config directories appear
	retryFileCh := make(chan event, 100)

	// Start config directory watcher goroutine
	go app.watchConfigDir(ctx, configWatcher, pendingFiles, retryFileCh)

	// Recursively watch the data directory and inventory its files
	walkDirFunc := func(path string, d fs.DirEntry, err error) error {
		path = filepath.Clean(filepath.Join(app.config.logsDir, string(filepath.Separator), path))
		if err != nil {
			return fmt.Errorf("walk directory %s: %w", path, err)
		}
		fileInfo, err := d.Info()
		if err != nil {
			return fmt.Errorf("get file info for %s: %w", path, err)
		}
		fid, err := getFileId(path)
		if err != nil {
			return fmt.Errorf("get file id for %s: %w", path, err)
		}
		seekInfoCache.Store(fid, &tail.SeekInfo{Offset: fileInfo.Size(), Whence: io.SeekStart})
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	}
	err = fs.WalkDir(os.DirFS(app.config.logsDir), ".", walkDirFunc)
	if err != nil {
		app.logger.Fatal().Str("component", "logprocessor").Err(err)
	}

	// Capture the fileId for each event
	eventsCounter := metrics.NewCounter("tslogs_events_total")
	events := pipeline.TransformerFunc(ctx, w.Events, addFileId(eventsCounter))

	// Filter the events to just the actionable ones
	actionableEventsCounter := metrics.NewCounter("tslogs_actionable_events_total")
	actionableEvents, _ := pipeline.FilterFunc(ctx, events, filterActionableEvents(app, tailing, actionableEventsCounter))

	// Separate the events into files and directories
	files, dirs := pipeline.FilterFunc(ctx, actionableEvents, separateFilesAndDirs)

	// Handle directories - scan for existing files and emit synthetic events
	dirsCounter := metrics.NewCounter("tslogs_dir_events_total")
	dirFiles := pipeline.DecomposerFunc(ctx, dirs, handleDirs(app, w, dirsCounter))

	// Merge directory-discovered files with directly-detected files and retry channel
	allFiles := pipeline.Merge(ctx, files, dirFiles, retryFileCh)

	// Handle files
	filesCounter := metrics.NewCounter("tslogs_file_events_total")
	tails := pipeline.TransformerFunc(ctx, allFiles, handleFiles(app, tailing, seekInfoCache, pendingFiles, filesCounter))

	// Filter empty tails
	tails, _ = pipeline.FilterFunc(ctx, tails, func(ctx context.Context, t tailedFile) bool {
		return t.Tail != nil
	})

	// For each tailed file, create a goroutine to receive, add metadata, and send its lines to the output channel
	tailsCounter := metrics.NewCounter("tslogs_tails_total")
	lineChs := pipeline.TransformerFunc(ctx, tails, lineProcessor(tailing, seekInfoCache, app, tailsCounter))

	// Bridge lines from the line channels
	lines := pipeline.Bridge(ctx, lineChs)

	return lines
}

func addFileId(counter *metrics.Counter) func(context.Context, fsnotify.Event) event {
	return func(ctx context.Context, e fsnotify.Event) event {
		counter.Inc()
		fid, err := getFileId(e.Name)
		if err != nil {
			return event{e, 0}
		}
		return event{e, fid}
	}
}

func filterActionableEvents(app *application, tailing *sync.Map, counter *metrics.Counter) func(context.Context, event) bool {
	skipFiles := app.config.skipFiles
	return func(ctx context.Context, e event) bool {
		if _, ok := tailing.Load(e.fileId); ok {
			return false
		}
		if !(e.Op&fsnotify.Create == fsnotify.Create || e.Op&fsnotify.Write == fsnotify.Write) {
			// Not a create or write event, so not actionable
			return false
		}
		fileInfo, err := os.Stat(e.Name)
		if err != nil {
			// We can't stat it, so it's not actionable
			return false
		}
		if !(fileInfo.IsDir() || fileInfo.Mode().IsRegular()) {
			// Neither a directory nor a regular file, so not actionable
			return false
		}
		for _, skipFile := range skipFiles {
			if strings.Contains(e.Name, skipFile) {
				return false
			}
		}
		counter.Inc()
		return true
	}
}

func separateFilesAndDirs(_ context.Context, e event) bool {
	fileInfo, err := os.Stat(e.Name)
	if err != nil {
		return false
	}
	if fileInfo.IsDir() {
		return false
	}
	return true
}

func handleDirs(app *application, w *fsnotify.Watcher, counter *metrics.Counter) func(event) []event {
	return func(e event) []event {
		counter.Inc()
		if err := w.Add(e.Name); err != nil {
			app.logger.Warn().Err(err).Str("directory", e.Name).Msg("failed to add directory to watcher")
			return nil
		}

		// Scan directory for existing files and return synthetic events
		entries, err := os.ReadDir(e.Name)
		if err != nil {
			app.logger.Warn().Err(err).Str("directory", e.Name).Msg("failed to read directory")
			return nil
		}

		var fileEvents []event
		for _, entry := range entries {
			if entry.IsDir() {
				continue // Only handle files; nested dirs will get their own events
			}
			filePath := filepath.Join(e.Name, entry.Name())
			fid, err := getFileId(filePath)
			if err != nil {
				app.logger.Debug().Err(err).Str("file", filePath).Msg("failed to get file ID")
				continue
			}
			fileEvents = append(fileEvents, event{
				Event:  fsnotify.Event{Name: filePath, Op: fsnotify.Create},
				fileId: fid,
			})
		}
		return fileEvents
	}
}

func handleFiles(app *application, tailing *sync.Map, seekInfoCache *sync.Map, pendingFiles *sync.Map, counter *metrics.Counter) func(context.Context, event) tailedFile {
	return func(ctx context.Context, e event) tailedFile {
		if _, ok := tailing.Load(e.fileId); ok { // If we're already tailing this file, skip this event
			return tailedFile{}
		}
		seekInfo, ok := seekInfoCache.Load(e.fileId) // Load the seekInfo from the cache
		if !ok {                                     // If it's not in the cache, it's a new file
			seekInfo = &tail.SeekInfo{Offset: 0, Whence: io.SeekStart} // so we'll tail from the beginning
			seekInfoCache.Store(e.fileId, seekInfo)                    // and store the seekInfo in the cache
		}
		c := tail.Config{
			Location: seekInfo.(*tail.SeekInfo),
			Follow:   true,
			Logger:   tail.DiscardingLogger,
		}
		t, err := tail.TailFile(e.Name, c)
		if err != nil {
			app.logger.Err(err).Str("filename", e.Name).Int64("fileid", int64(e.fileId)).Msg("could not tail file. skipping")
			return tailedFile{}
		}
		processName := getProcessName(e.Name, app.config.logsDir)
		processId := getProcessId(filepath.Base(e.Name))
		component := getComponent(filepath.Base(e.Name))
		counter.Inc()
		instance, err := process.For(processId, processName, app.config.configDir)
		if err != nil {
			// If config directory not found, add to pending files for retry when config appears
			if errors.Is(err, process.ErrConfigDirNotFound) || errors.Is(err, process.ErrConfigFileNotFound) {
				pendingFiles.Store(e.fileId, e)
				pendingFilesCounter := metrics.GetOrCreateCounter("tslogs_pending_files_total")
				pendingFilesCounter.Inc()
				app.logger.Info().
					Str("filename", t.Filename).
					Int64("fileid", int64(e.fileId)).
					Str("processname", processName).
					Int64("processid", int64(processId)).
					Str("configdir", app.config.configDir).
					Msg("config not found for process instance. queued for retry when config appears")
				t.Cleanup()
				return tailedFile{}
			}
			app.logger.Err(err).
				Str("filename", t.Filename).
				Int64("fileid", int64(e.fileId)).
				Str("processname", processName).
				Int64("processid", int64(processId)).
				Str("configdir", app.config.configDir).
				Msg("could not get process instance. skipping")
			return tailedFile{}
		}
		logFormat := instance.GetLogFormat(e.Name)
		tailing.Store(e.fileId, t)
		return tailedFile{t, e.fileId, processName, processId, component, logFormat}
	}
}

func lineProcessor(tailing *sync.Map, seekInfoCache *sync.Map, app *application, counter *metrics.Counter) func(ctx context.Context, t tailedFile) <-chan line {
	counters := make(map[string]*metrics.Counter)
	return func(ctx context.Context, t tailedFile) <-chan line {
		linesCounter := metrics.GetOrCreateCounter(fmt.Sprintf("tslogs_lines_received_total{filename=%q, fileid=%q}", t.Filename, t.fileId))
		lineCh := make(chan line)
		go func(ctx context.Context, lineCh chan<- line) {
			defer func() {
				tailing.Delete(t.fileId)
				counter.Dec()
			}()
			counter.Inc()
			linesCounter.Set(0)
			path := t.Filename
			fid := t.fileId
			var re *regexp.Regexp
			var err error
			if t.logFormat != "json" && t.logFormat != "" {
				re, err = regexp.Compile(t.logFormat)
				if err != nil {
					app.logger.Err(err).Str("filename", t.Filename).Int64("fileid", int64(t.fileId)).Msg("could not compile parser. skipping")
					return
				}
			}
			newEntry := func(line string) bool {
				if t.logFormat == "json" {
					if len(line) > 0 && line[0] == '{' {
						return true
					}
					return false
				}
				if t.logFormat == "" {
					return true
				}
				if re != nil {
					return re.MatchString(line)
				}
				return true
			}
			completeEntry := func(line string) completeness {
				if t.logFormat == "json" {
					if line[0] == '{' && line[len(line)-1] == '}' {
						return complete
					}
					return incomplete
				}
				if t.logFormat == "" {
					return complete
				}
				return unknownCompleteness
			}
			parse := func(text string) string {
				if t.logFormat == "json" {
					return text
				}
				if t.logFormat == "" {
					return text
				}
				if re != nil {
					matches := re.FindStringSubmatch(text)
					if len(matches) == 0 {
						return text
					}
					result := make(map[string]string)
					for i, name := range re.SubexpNames() {
						if i != 0 && name != "" {
							result[name] = matches[i]
						}
					}
					b, err := json.Marshal(result)
					if err != nil {
						return text
					}
					return string(b)
				}
				return text
			}
			sendAccumulatedLines := func(l []*tail.Line) {
				combinedLine := &tail.Line{
					Num:      l[0].Num,
					SeekInfo: l[len(l)-1].SeekInfo,
					Time:     l[0].Time,
					Err:      l[0].Err,
				}
				var rawText string
				for i, v := range l {
					if i == 0 {
						rawText += v.Text
					} else {
						rawText += "\n" + v.Text
					}
				}
				if app.config.parse {
					combinedLine.Text = parse(rawText)
				} else {
					combinedLine.Text = rawText
				}
				output := line{combinedLine, path, fid, t.processName, t.processId, t.component}
				lineCh <- output
				logEntryCounterName := fmt.Sprintf("tslogs_entries_total{process=%q, node=%q, component=%q, level=%q}", t.processName, app.config.node, t.component, output.Level())
				if logEntryCounter, ok := counters[logEntryCounterName]; !ok {
					logEntryCounter = metrics.NewCounter(logEntryCounterName)
					counters[logEntryCounterName] = logEntryCounter
					logEntryCounter.Inc()
				} else {
					logEntryCounter.Inc()
				}
			}
			lines := make([]*tail.Line, 0)
			for {
				select {
				case <-ctx.Done():
					if len(lines) > 0 {
						sendAccumulatedLines(lines)
					}
					return
				case l, ok := <-t.Lines:
					if !ok {
						return
					}
					seekInfoCache.Store(fid, &l.SeekInfo)
					// Metrics
					{
						linesCounter.Inc()
						if l.Err != nil {
							errorCounterName := fmt.Sprintf("tslogs_lines_processed_error_total{process=%q, node=%q, component=%q}", t.processName, app.config.node, t.component)
							if errorCounter, ok := counters[errorCounterName]; !ok {
								errorCounter = metrics.NewCounter(errorCounterName)
								counters[errorCounterName] = errorCounter
								errorCounter.Inc()
							} else {
								errorCounter.Inc()
							}
						}

					}

					if !newEntry(l.Text) {
						lines = append(lines, l)
						continue
					}
					// It is a new entry
					// Do we have any accumulated lines to send?
					if len(lines) > 0 {
						sendAccumulatedLines(lines)
						lines = make([]*tail.Line, 0)
					}
					// If it's a complete entry, no need to accumulate lines. Send it!
					if completeEntry(l.Text) == complete {
						output := line{l, path, fid, t.processName, t.processId, t.component}
						lineCh <- output
						continue
					}
					// Otherwise, accumulate lines
					lines = append(lines, l)

				case <-time.After(time.Minute * 5):
					t.Cleanup()
					return
				}
			}
		}(ctx, lineCh)
		return lineCh
	}
}

type completeness int

const (
	incomplete completeness = iota
	complete
	unknownCompleteness
)

// watchConfigDir monitors the config directory for new process instance directories.
// When a new config directory appears (e.g., vizqlserver_1/), it checks if there are
// pending log files waiting for that config and triggers a retry.
func (app *application) watchConfigDir(ctx context.Context, watcher *fsnotify.Watcher, pendingFiles *sync.Map, retryCh chan<- event) {
	defer watcher.Close()

	configDir := app.config.configDir
	if configDir == "" {
		app.logger.Warn().Msg("config directory not specified, dynamic process discovery disabled")
		return
	}

	// Add the config directory to the watcher
	if err := watcher.Add(configDir); err != nil {
		app.logger.Err(err).Str("configdir", configDir).Msg("could not watch config directory")
		return
	}
	app.logger.Info().Str("configdir", configDir).Msg("watching config directory for new process instances")

	configDirDiscoveryCounter := metrics.GetOrCreateCounter("tslogs_config_dir_discovery_total")

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only care about Create events for directories
			if e.Op&fsnotify.Create != fsnotify.Create {
				continue
			}
			fileInfo, err := os.Stat(e.Name)
			if err != nil || !fileInfo.IsDir() {
				continue
			}

			dirName := filepath.Base(e.Name)
			app.logger.Info().
				Str("directory", dirName).
				Str("path", e.Name).
				Msg("detected new config directory")
			configDirDiscoveryCounter.Inc()

			// Check if there are pending files for this process instance
			// Recompute processName_processId from each file's metadata to find matches
			// Collect matching entries first to avoid blocking the Range callback
			var matchingEntries []struct {
				fileId fileId
				event  event
			}
			pendingFiles.Range(func(key, value interface{}) bool {
				fid := key.(fileId)
				pendingEvent := value.(event)

				// Recompute processName and processId from the file path
				processName := getProcessName(pendingEvent.Name, app.config.logsDir)
				processId := getProcessId(filepath.Base(pendingEvent.Name))
				pendingKey := fmt.Sprintf("%s_%d", processName, processId)

				// Check if this config directory matches the pending process
				// Must be exact match OR match with underscore suffix (for version suffixes like vizqlserver_1_abc123)
				// This prevents vizqlserver_1 from matching vizqlserver_10
				if dirName == pendingKey || strings.HasPrefix(dirName, pendingKey+"_") {
					matchingEntries = append(matchingEntries, struct {
						fileId fileId
						event  event
					}{fid, pendingEvent})
				}
				return true
			})

			// Process matching entries after Range completes
			if len(matchingEntries) > 0 {
				// Wait once for config files to be written
				time.Sleep(500 * time.Millisecond)
			}
			for _, entry := range matchingEntries {
				app.logger.Info().
					Str("filename", entry.event.Name).
					Int64("fileid", int64(entry.event.fileId)).
					Str("configdir", dirName).
					Msg("retrying log file after config directory appeared")

				// Remove from pending and send to retry channel
				pendingFiles.Delete(entry.fileId)
				select {
				case retryCh <- entry.event:
				default:
					app.logger.Warn().
						Str("filename", entry.event.Name).
						Msg("retry channel full, could not queue file for retry")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			app.logger.Err(err).Msg("config directory watcher error")
		}
	}
}
