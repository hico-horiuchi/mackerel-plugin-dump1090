# mackerel-plugin-dump1090

dump1090-fa custom metrics plugin for mackerel.io agent.

This plugin collects metrics from dump1090 (dump1090-mutability, dump1090-fa, or readsb) and reports them to Mackerel. It provides equivalent functionality to the [dump1090-exporter](https://github.com/claws/dump1090-exporter) Prometheus exporter.

## Usage

```
mackerel-plugin-dump1090 [options]
```

### Options

- `--resource-path <path>` : Resource path (URL or file path) to dump1090 data (default: `http://localhost:8080/data`)
- `--latitude <float>` : Receiver latitude for range calculation (optional)
- `--longitude <float>` : Receiver longitude for range calculation (optional)
- `--metric-key-prefix <prefix>` : Metric key prefix (default: `dump1090`)
- `--version` : Show version

## Example

### Monitor remote dump1090 instance

```
$ ./mackerel-plugin-dump1090 \
  --resource-path=http://192.168.1.201:8080/data \
  --latitude=-34.9285 \
  --longitude=138.6007
```

### Monitor local dump1090-fa instance

```
$ ./mackerel-plugin-dump1090 \
  --resource-path=/run/dump1090-fa/ \
  --latitude=35.6762 \
  --longitude=139.6503
```

## Metrics

The plugin collects the following metrics:

### Aircraft Metrics

- **observed** : Number of aircraft recently observed
- **with_position** : Number of aircraft with position
- **with_mlat** : Number of aircraft with multilateration
- **max_range** : Maximum range of observed aircraft (km)
- **messages_total** : Total number of Mode-S messages

### Statistics Metrics (from last 1 minute)

- **stats_messages** : Number of Mode-S messages processed
- **stats_cpr_*** : CPR (Compact Position Reporting) statistics
- **stats_cpu_*** : CPU usage statistics (background, demod, reader)
- **stats_local_signal** : Signal strength (dBFS)
- **stats_local_peak_signal** : Peak signal strength (dBFS)
- **stats_local_noise** : Noise level (dBFS)
- **stats_local_*** : Local receiver statistics
- **stats_tracks_*** : Track statistics

## Install

```
mkr plugin install hico-horiuchi/mackerel-plugin-dump1090
```

## Add mackerel-agent.conf

For remote dump1090 instance:

```
[plugin.metrics.dump1090]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-dump1090 --resource-path=http://localhost:8080/data --latitude=YOUR_LAT --longitude=YOUR_LON"
```

For local dump1090-fa instance:

```
[plugin.metrics.dump1090]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-dump1090 --resource-path=/run/dump1090-fa/"
```

## Requirements

- dump1090-mutability, dump1090-fa, or readsb running and accessible
- The dump1090 instance should expose JSON data files (receiver.json, aircraft.json, stats.json)

## Author

[hico-horiuchi](https://github.com/hico-horiuchi/)
