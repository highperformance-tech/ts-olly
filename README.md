# ts-olly

A real-time log observer for Tableau Server. Tails log files, parses structured log formats (log4j, log4j2, httpd), and outputs unified JSON logs suitable for ingestion into log aggregation systems.

## Features

- Real-time log tailing with automatic discovery of new log files
- Parses multiple log formats:
  - log4j XML configuration
  - log4j2 XML configuration
  - Apache httpd custom log formats
  - JSON logs (passthrough)
- Outputs structured JSON logs via zerolog
- Exposes Prometheus metrics endpoint
- Graceful shutdown handling
- Docker support with minimal distroless image

## Installation

### From Source

```bash
go install github.com/highperformance-tech/ts-olly/cmd/ts-olly@latest
```

### Build from Source

```bash
git clone https://github.com/highperformance-tech/ts-olly.git
cd ts-olly
make build
```

### Docker

```bash
docker pull ghcr.io/highperformance-tech/ts-olly:latest
```

## Usage

```bash
ts-olly [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `2112` | Port for metrics endpoint |
| `-env` | `development` | Environment (development\|staging\|production) |
| `-node` | | Tableau cluster node ID (e.g., node1, node2) |
| `-logsdir` | | Path to Tableau Server logs directory |
| `-configdir` | | Path to Tableau Server config directory |
| `-parse` | `false` | Parse recognizable log lines into structured JSON |
| `-read-existing-logs` | `false` | Read existing log content on startup |

### Example

```bash
ts-olly \
  -node node1 \
  -logsdir /var/opt/tableau/tableau_server/data/tabsvc/logs \
  -configdir /var/opt/tableau/tableau_server/data/tabsvc/config \
  -port 2112
```

### Docker

```bash
docker run -d \
  -v /var/opt/tableau/tableau_server/data/tabsvc/logs:/logs:ro \
  -v /var/opt/tableau/tableau_server/data/tabsvc/config:/config:ro \
  -p 2112:2112 \
  ghcr.io/highperformance-tech/ts-olly:latest \
  -node node1 \
  -logsdir /logs \
  -configdir /config
```

## Output

ts-olly outputs JSON-formatted log lines to stdout. Each line includes:

- `filename` - Source log file path
- `fileid` - Unique file identifier
- `process` - Tableau process name
- `processid` - Process instance ID
- `line` - Line number in source file
- `offset` - Byte offset in source file
- `level` - Log level (when parseable)
- `component` - Log component/logger name
- `message` - Log message content
- `node` - Cluster node identifier
- `ts` - Timestamp

## Metrics

Prometheus metrics are exposed at `http://localhost:<port>/metrics`.

## License

MIT License - see [LICENSE](LICENSE) for details.
