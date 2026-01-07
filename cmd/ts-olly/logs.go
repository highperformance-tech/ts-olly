package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/VictoriaMetrics/metrics"
	"github.com/fsnotify/fsnotify"
	"github.com/highperformance-tech/ts-olly/cmd/ts-olly/process"
	"github.com/highperformance-tech/ts-olly/internal/pipeline"
	"github.com/nxadm/tail"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

	// Initialize watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		app.logger.Fatal().Err(err)
	}
	app.watcher = w

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

	//Handle directories
	dirsCounter := metrics.NewCounter("tslogs_dir_events_total")
	pipeline.SinkFunc(ctx, dirs, handleDirs(w, dirsCounter))

	// Handle files
	filesCounter := metrics.NewCounter("tslogs_file_events_total")
	tails := pipeline.TransformerFunc(ctx, files, handleFiles(app, tailing, seekInfoCache, filesCounter))

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

func handleDirs(w *fsnotify.Watcher, counter *metrics.Counter) func(context.Context, event) {
	return func(ctx context.Context, e event) {
		counter.Inc()
		_ = w.Add(e.Name) // If there was an error adding the directory, we'd return false anyway
	}
}

func handleFiles(app *application, tailing *sync.Map, seekInfoCache *sync.Map, counter *metrics.Counter) func(context.Context, event) tailedFile {
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
