// Package logger provides custom logging utilities for the application.
package logger

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// PipeEncoder is a custom zapcore.Encoder that outputs fields in pipe-delimited format.
// Instead of JSON format like {"key": "value"}, it outputs: key="value" | key=123
type PipeEncoder struct {
	zapcore.Encoder
	pool      buffer.Pool
	separator string
	fields    []string
}

// NewPipeEncoder creates a new PipeEncoder with the specified separator.
func NewPipeEncoder(separator string) *PipeEncoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		EncodeTime:       zapcore.ISO8601TimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: separator,
	}

	return &PipeEncoder{
		Encoder:   zapcore.NewConsoleEncoder(encoderConfig),
		pool:      buffer.NewPool(),
		separator: separator,
		fields:    make([]string, 0),
	}
}

// Clone creates a copy of the encoder.
func (e *PipeEncoder) Clone() zapcore.Encoder {
	return &PipeEncoder{
		Encoder:   e.Encoder.Clone(),
		pool:      e.pool,
		separator: e.separator,
		fields:    append([]string{}, e.fields...),
	}
}

// EncodeEntry encodes a log entry with pipe-delimited fields.
func (e *PipeEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// Create a clone to collect fields without modifying the original
	clone := e.Clone().(*PipeEncoder)
	clone.fields = make([]string, 0)

	// Process each field to collect formatted strings
	for _, field := range fields {
		clone.addField(field)
	}

	// Encode the base entry (timestamp | LEVEL | message) without fields
	buf, err := clone.Encoder.EncodeEntry(entry, nil)
	if err != nil {
		return nil, err
	}

	// If we have fields, append them in pipe-delimited format
	if len(clone.fields) > 0 {
		// Remove the trailing newline to append fields
		content := buf.String()
		content = strings.TrimSuffix(content, "\n")

		// Create new buffer with fields appended
		newBuf := e.pool.Get()
		newBuf.AppendString(content)

		for _, f := range clone.fields {
			newBuf.AppendString(e.separator)
			newBuf.AppendString(f)
		}
		newBuf.AppendString("\n")

		buf.Free()
		return newBuf, nil
	}

	return buf, nil
}

// addField processes a single field and adds it to the fields slice.
func (e *PipeEncoder) addField(field zapcore.Field) {
	switch field.Type {
	case zapcore.StringType:
		e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, field.String))

	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		e.fields = append(e.fields, fmt.Sprintf("%s=%d", field.Key, field.Integer))

	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		e.fields = append(e.fields, fmt.Sprintf("%s=%d", field.Key, field.Integer))

	case zapcore.Float64Type:
		e.fields = append(e.fields, fmt.Sprintf("%s=%.2f", field.Key, float64(field.Integer)))

	case zapcore.Float32Type:
		e.fields = append(e.fields, fmt.Sprintf("%s=%.2f", field.Key, float64(field.Integer)))

	case zapcore.BoolType:
		val := "false"
		if field.Integer == 1 {
			val = "true"
		}
		e.fields = append(e.fields, fmt.Sprintf("%s=%s", field.Key, val))

	case zapcore.TimeType:
		t := time.Unix(0, field.Integer)
		if field.Interface != nil {
			t = t.In(field.Interface.(*time.Location))
		}
		e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, t.Format(time.RFC3339)))

	case zapcore.DurationType:
		d := time.Duration(field.Integer)
		e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, d.String()))

	case zapcore.ErrorType:
		if err, ok := field.Interface.(error); ok {
			e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, err.Error()))
		}

	case zapcore.StringerType:
		if stringer, ok := field.Interface.(fmt.Stringer); ok {
			e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, stringer.String()))
		}

	default:
		// For complex types, use the default string representation
		if field.Interface != nil {
			e.fields = append(e.fields, fmt.Sprintf("%s=%q", field.Key, fmt.Sprintf("%v", field.Interface)))
		}
	}
}

// NewPipeEncoderCore creates a zapcore.Core with the PipeEncoder.
func NewPipeEncoderCore(level zapcore.Level) zapcore.Core {
	encoder := NewPipeEncoder(" | ")
	return zapcore.NewCore(
		encoder,
		zapcore.AddSync(zapcore.Lock(zapcore.AddSync(createStdoutSyncer()))),
		level,
	)
}

// createStdoutSyncer creates a write syncer for stdout.
func createStdoutSyncer() zapcore.WriteSyncer {
	return zapcore.AddSync(&stdoutWriter{})
}

// stdoutWriter is a simple writer that writes to stdout.
type stdoutWriter struct{}

func (w *stdoutWriter) Write(p []byte) (n int, err error) {
	return fmt.Print(string(p))
}

// NewLogger creates a new zap.Logger with the PipeEncoder.
func NewLogger() *zap.Logger {
	core := NewPipeEncoderCore(zapcore.InfoLevel)
	return zap.New(core)
}

// NewLoggerWithLevel creates a new zap.Logger with the PipeEncoder and specified level.
func NewLoggerWithLevel(level zapcore.Level) *zap.Logger {
	core := NewPipeEncoderCore(level)
	return zap.New(core)
}
