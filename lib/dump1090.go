package mpdump1090

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// Dump1090Plugin represents the plugin
type Dump1090Plugin struct {
	prefix       string
	resourcePath string
	latitude     float64
	longitude    float64
}

// ReceiverData represents the receiver.json structure
type ReceiverData struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

// AircraftData represents the aircraft.json structure
type AircraftData struct {
	Aircraft     []Aircraft `json:"aircraft"`
	Messages     int64      `json:"messages"`
	Now          float64    `json:"now"`
}

// Aircraft represents individual aircraft data
type Aircraft struct {
	Hex      string   `json:"hex"`
	Flight   string   `json:"flight,omitempty"`
	Lat      *float64 `json:"lat,omitempty"`
	Lon      *float64 `json:"lon,omitempty"`
	Track    *float64 `json:"track,omitempty"`
	Mlat     []string `json:"mlat,omitempty"`
	Messages int64    `json:"messages"`
}

// StatsData represents the stats.json structure
type StatsData struct {
	Latest   StatsGroup `json:"latest"`
	Last1Min StatsGroup `json:"last1min"`
	Last5Min StatsGroup `json:"last5min"`
	Last15Min StatsGroup `json:"last15min"`
	Total    StatsGroup `json:"total"`
}

// StatsGroup represents a time period's statistics
type StatsGroup struct {
	Start    float64         `json:"start,omitempty"`
	End      float64         `json:"end,omitempty"`
	Messages *int64          `json:"messages,omitempty"`
	CPR      *CPRStats       `json:"cpr,omitempty"`
	CPU      *CPUStats       `json:"cpu,omitempty"`
	Local    *LocalStats     `json:"local,omitempty"`
	Remote   *RemoteStats    `json:"remote,omitempty"`
	Tracks   *TracksStats    `json:"tracks,omitempty"`
}

// CPRStats represents CPR statistics
type CPRStats struct {
	Airborne              *int64 `json:"airborne,omitempty"`
	Surface               *int64 `json:"surface,omitempty"`
	Filtered              *int64 `json:"filtered,omitempty"`
	GlobalBad             *int64 `json:"global_bad,omitempty"`
	GlobalOk              *int64 `json:"global_ok,omitempty"`
	GlobalRange           *int64 `json:"global_range,omitempty"`
	GlobalSkipped         *int64 `json:"global_skipped,omitempty"`
	GlobalSpeed           *int64 `json:"global_speed,omitempty"`
	LocalAircraftRelative *int64 `json:"local_aircraft_relative,omitempty"`
	LocalOk               *int64 `json:"local_ok,omitempty"`
	LocalRange            *int64 `json:"local_range,omitempty"`
	LocalReceiverRelative *int64 `json:"local_receiver_relative,omitempty"`
	LocalSkipped          *int64 `json:"local_skipped,omitempty"`
	LocalSpeed            *int64 `json:"local_speed,omitempty"`
}

// CPUStats represents CPU statistics
type CPUStats struct {
	Background *int64 `json:"background,omitempty"`
	Demod      *int64 `json:"demod,omitempty"`
	Reader     *int64 `json:"reader,omitempty"`
}

// LocalStats represents local receiver statistics
type LocalStats struct {
	Accepted         []int64  `json:"accepted,omitempty"`
	Signal           *float64 `json:"signal,omitempty"`
	PeakSignal       *float64 `json:"peak_signal,omitempty"`
	Noise            *float64 `json:"noise,omitempty"`
	StrongSignals    *int64   `json:"strong_signals,omitempty"`
	Bad              *int64   `json:"bad,omitempty"`
	Modes            *int64   `json:"modes,omitempty"`
	ModeAC           *int64   `json:"modeac,omitempty"`
	SamplesDropped   *int64   `json:"samples_dropped,omitempty"`
	SamplesProcessed *int64   `json:"samples_processed,omitempty"`
	UnknownICAO      *int64   `json:"unknown_icao,omitempty"`
}

// RemoteStats represents remote statistics
type RemoteStats struct {
	Accepted    []int64 `json:"accepted,omitempty"`
	Bad         *int64  `json:"bad,omitempty"`
	ModeAC      *int64  `json:"modeac,omitempty"`
	Modes       *int64  `json:"modes,omitempty"`
	UnknownICAO *int64  `json:"unknown_icao,omitempty"`
}

// TracksStats represents tracks statistics
type TracksStats struct {
	All           *int64 `json:"all,omitempty"`
	SingleMessage *int64 `json:"single_message,omitempty"`
}

// MetricKeyPrefix returns the prefix for metric keys
func (d Dump1090Plugin) MetricKeyPrefix() string {
	if d.prefix == "" {
		d.prefix = "dump1090"
	}
	return d.prefix
}

// GraphDefinition returns graph definitions
func (d Dump1090Plugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(d.prefix)
	return map[string]mp.Graphs{
		"aircraft": {
			Label: labelPrefix + " Aircraft",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "observed", Label: "Recent Aircraft Observed"},
				{Name: "with_position", Label: "Recent Aircraft With Position"},
				{Name: "with_mlat", Label: "Recent Aircraft With Multilateration"},
			},
		},
		"range": {
			Label: labelPrefix + " Range",
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "max_range", Label: "Max Range (km)"},
			},
		},
		"messages": {
			Label: labelPrefix + " Messages",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "messages_total", Label: "Messages Total"},
			},
		},
		"stats.messages": {
			Label: labelPrefix + " Stats Messages",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_messages", Label: "Messages"},
			},
		},
		"stats.cpr": {
			Label: labelPrefix + " Stats CPR",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_cpr_airborne", Label: "CPR Airborne"},
				{Name: "stats_cpr_surface", Label: "CPR Surface"},
				{Name: "stats_cpr_global_ok", Label: "CPR Global OK"},
				{Name: "stats_cpr_local_ok", Label: "CPR Local OK"},
			},
		},
		"stats.cpu": {
			Label: labelPrefix + " Stats CPU",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_cpu_background", Label: "CPU Background (ms)"},
				{Name: "stats_cpu_demod", Label: "CPU Demod (ms)"},
				{Name: "stats_cpu_reader", Label: "CPU Reader (ms)"},
			},
		},
		"stats.local.signal": {
			Label: labelPrefix + " Stats Local Signal",
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "stats_local_signal", Label: "Signal Strength (dBFS)"},
				{Name: "stats_local_peak_signal", Label: "Peak Signal Strength (dBFS)"},
				{Name: "stats_local_noise", Label: "Noise Level (dBFS)"},
			},
		},
		"stats.local.messages": {
			Label: labelPrefix + " Stats Local Messages",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_local_accepted", Label: "Accepted"},
				{Name: "stats_local_bad", Label: "Bad"},
				{Name: "stats_local_modes", Label: "Mode S Preambles"},
				{Name: "stats_local_strong_signals", Label: "Strong Signals"},
			},
		},
		"stats.local.samples": {
			Label: labelPrefix + " Stats Local Samples",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_local_samples_processed", Label: "Samples Processed"},
				{Name: "stats_local_samples_dropped", Label: "Samples Dropped"},
			},
		},
		"stats.tracks": {
			Label: labelPrefix + " Stats Tracks",
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "stats_tracks_all", Label: "All Tracks"},
				{Name: "stats_tracks_single_message", Label: "Single Message Tracks"},
			},
		},
	}
}

// fetchJSON fetches and parses JSON from a file or URL
func (d Dump1090Plugin) fetchJSON(path string, v interface{}) error {
	var data []byte
	var err error

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// Fetch from HTTP
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(path)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error: %d", resp.StatusCode)
		}
		
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	} else {
		// Read from file
		data, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	}

	return json.Unmarshal(data, v)
}

// calculateDistance calculates the distance between two points in kilometers
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // Earth radius in kilometers

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// FetchMetrics fetches metrics from dump1090
func (d Dump1090Plugin) FetchMetrics() (map[string]float64, error) {
	result := make(map[string]float64)

	// Fetch aircraft data
	aircraftPath := filepath.Join(d.resourcePath, "aircraft.json")
	if !strings.HasPrefix(d.resourcePath, "http://") && !strings.HasPrefix(d.resourcePath, "https://") {
		// For file paths, use filepath.Join
		aircraftPath = filepath.Join(d.resourcePath, "aircraft.json")
	} else {
		// For URLs, use string concatenation
		aircraftPath = strings.TrimSuffix(d.resourcePath, "/") + "/aircraft.json"
	}

	var aircraftData AircraftData
	if err := d.fetchJSON(aircraftPath, &aircraftData); err == nil {
		observed := len(aircraftData.Aircraft)
		withPosition := 0
		withMlat := 0
		maxRange := 0.0

		for _, aircraft := range aircraftData.Aircraft {
			if aircraft.Lat != nil && aircraft.Lon != nil {
				withPosition++
				
				// Calculate range if we have receiver position
				if d.latitude != 0 && d.longitude != 0 {
					distance := calculateDistance(d.latitude, d.longitude, *aircraft.Lat, *aircraft.Lon)
					if distance > maxRange {
						maxRange = distance
					}
				}
			}
			if len(aircraft.Mlat) > 0 {
				withMlat++
			}
		}

		result["observed"] = float64(observed)
		result["with_position"] = float64(withPosition)
		result["with_mlat"] = float64(withMlat)
		result["max_range"] = maxRange
		result["messages_total"] = float64(aircraftData.Messages)
	}

	// Fetch stats data
	statsPath := filepath.Join(d.resourcePath, "stats.json")
	if !strings.HasPrefix(d.resourcePath, "http://") && !strings.HasPrefix(d.resourcePath, "https://") {
		statsPath = filepath.Join(d.resourcePath, "stats.json")
	} else {
		statsPath = strings.TrimSuffix(d.resourcePath, "/") + "/stats.json"
	}

	var statsData StatsData
	if err := d.fetchJSON(statsPath, &statsData); err == nil {
		// Use last1min stats as the primary metrics (similar to the exporter)
		stats := statsData.Last1Min

		if stats.Messages != nil {
			result["stats_messages"] = float64(*stats.Messages)
		}

		// CPR stats
		if stats.CPR != nil {
			if stats.CPR.Airborne != nil {
				result["stats_cpr_airborne"] = float64(*stats.CPR.Airborne)
			}
			if stats.CPR.Surface != nil {
				result["stats_cpr_surface"] = float64(*stats.CPR.Surface)
			}
			if stats.CPR.GlobalOk != nil {
				result["stats_cpr_global_ok"] = float64(*stats.CPR.GlobalOk)
			}
			if stats.CPR.LocalOk != nil {
				result["stats_cpr_local_ok"] = float64(*stats.CPR.LocalOk)
			}
		}

		// CPU stats
		if stats.CPU != nil {
			if stats.CPU.Background != nil {
				result["stats_cpu_background"] = float64(*stats.CPU.Background)
			}
			if stats.CPU.Demod != nil {
				result["stats_cpu_demod"] = float64(*stats.CPU.Demod)
			}
			if stats.CPU.Reader != nil {
				result["stats_cpu_reader"] = float64(*stats.CPU.Reader)
			}
		}

		// Local stats
		if stats.Local != nil {
			if stats.Local.Signal != nil {
				result["stats_local_signal"] = *stats.Local.Signal
			}
			if stats.Local.PeakSignal != nil {
				result["stats_local_peak_signal"] = *stats.Local.PeakSignal
			}
			if stats.Local.Noise != nil {
				result["stats_local_noise"] = *stats.Local.Noise
			}
			if stats.Local.StrongSignals != nil {
				result["stats_local_strong_signals"] = float64(*stats.Local.StrongSignals)
			}
			if stats.Local.Bad != nil {
				result["stats_local_bad"] = float64(*stats.Local.Bad)
			}
			if stats.Local.Modes != nil {
				result["stats_local_modes"] = float64(*stats.Local.Modes)
			}
			if stats.Local.SamplesProcessed != nil {
				result["stats_local_samples_processed"] = float64(*stats.Local.SamplesProcessed)
			}
			if stats.Local.SamplesDropped != nil {
				result["stats_local_samples_dropped"] = float64(*stats.Local.SamplesDropped)
			}
			// Sum up accepted array if present
			if len(stats.Local.Accepted) > 0 {
				var total int64
				for _, v := range stats.Local.Accepted {
					total += v
				}
				result["stats_local_accepted"] = float64(total)
			}
		}

		// Tracks stats
		if stats.Tracks != nil {
			if stats.Tracks.All != nil {
				result["stats_tracks_all"] = float64(*stats.Tracks.All)
			}
			if stats.Tracks.SingleMessage != nil {
				result["stats_tracks_single_message"] = float64(*stats.Tracks.SingleMessage)
			}
		}
	}

	return result, nil
}

// Do runs the plugin
func Do() {
	optPrefix := flag.String("metric-key-prefix", "dump1090", "Metric key prefix")
	optResourcePath := flag.String("resource-path", "http://localhost:8080/data", "Resource path (URL or file path)")
	optLatitude := flag.Float64("latitude", 0, "Receiver latitude")
	optLongitude := flag.Float64("longitude", 0, "Receiver longitude")
	optVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *optVersion {
		fmt.Printf(`Version: mackerel-plugin-dump1090 %s
Compiler: %s %s
`,
			VERSION,
			runtime.Compiler,
			runtime.Version())
		os.Exit(0)
	}

	mp.NewMackerelPlugin(&Dump1090Plugin{
		prefix:       *optPrefix,
		resourcePath: *optResourcePath,
		latitude:     *optLatitude,
		longitude:    *optLongitude,
	}).Run()
}
