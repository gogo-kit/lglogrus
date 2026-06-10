// Package lglogrus provides a sirupsen/logrus backend for gkit/log. It
// implements log.Driver and is selected via Config.Driver:
//
//	log.New(log.Config{Service: "order-service", Driver: lglogrus.Driver})
//
// It lives in its own module so logrus stays an opt-in dependency; the core log
// package defaults to a zero-dependency JSON driver. Output matches the core
// driver: a single line of contract JSON with UPPERCASE levels and RFC3339 (UTC)
// timestamps.
package lglogrus

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gogo-kit/log"
)

// Driver is a log.DriverFactory backed by logrus. Pass it to log.Config.Driver.
func Driver(opts log.DriverOptions) log.Driver {
	l := logrus.New()
	l.SetOutput(opts.Output)
	l.SetLevel(toLogrus(opts.Level))
	l.SetFormatter(&contractFormatter{opts: opts})
	return &driver{logger: l}
}

type driver struct {
	logger *logrus.Logger
}

func (d *driver) Write(level log.Level, msg string, fields map[string]any) {
	d.logger.WithFields(fields).Log(toLogrus(level), msg)
}

func (d *driver) Enabled(level log.Level) bool {
	return d.logger.IsLevelEnabled(toLogrus(level))
}

// Sync is a no-op: logrus writes synchronously.
func (d *driver) Sync() error { return nil }

// contractFormatter renders each entry as one line of JSON matching the ELK
// contract, replacing logrus' default JSON formatter. Key renames come from the
// driver options.
type contractFormatter struct {
	opts log.DriverOptions
}

func (f *contractFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	out := make(map[string]any, len(entry.Data)+5)

	// Data carries canonical keys; resolve each to its configured output name.
	for k, v := range entry.Data {
		out[f.opts.Key(k)] = v
	}

	out[f.opts.Key(log.KeyTimestamp)] = entry.Time.UTC().Format(time.RFC3339)
	out[f.opts.Key(log.KeyLevel)] = levelString(entry.Level)
	out[f.opts.Key(log.KeyService)] = f.opts.Service
	out[f.opts.Key(log.KeyEnvironment)] = f.opts.Environment
	out[f.opts.Key(log.KeyMessage)] = entry.Message

	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func toLogrus(l log.Level) logrus.Level {
	switch l {
	case log.DebugLevel:
		return logrus.DebugLevel
	case log.WarnLevel:
		return logrus.WarnLevel
	case log.ErrorLevel:
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}

func levelString(l logrus.Level) string {
	switch l {
	case logrus.WarnLevel:
		return "WARN"
	case logrus.PanicLevel:
		return "PANIC"
	case logrus.FatalLevel:
		return "FATAL"
	default:
		return strings.ToUpper(l.String())
	}
}
