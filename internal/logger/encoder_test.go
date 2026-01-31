package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// createTestLogger creates a logger that writes to a buffer for testing.
func createTestLogger(buf *bytes.Buffer) *zap.Logger {
	encoder := NewPipeEncoder(" | ")
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(buf),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}

func TestPipeEncoder_StringField(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Test message", zap.String("cluster", "production"))

	output := buf.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "Test message")
	assert.Contains(t, output, `cluster="production"`)
}

func TestPipeEncoder_IntField(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Test message", zap.Int("count", 42))

	output := buf.String()
	assert.Contains(t, output, "count=42")
	assert.NotContains(t, output, `count="42"`) // Should NOT have quotes
}

func TestPipeEncoder_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Filtering clusters",
		zap.Int("matched", 1),
		zap.Int("total", 19))

	output := buf.String()
	assert.Contains(t, output, "Filtering clusters")
	assert.Contains(t, output, "matched=1")
	assert.Contains(t, output, "total=19")
	assert.Contains(t, output, " | matched=1")
	assert.Contains(t, output, " | total=19")
}

func TestPipeEncoder_BoolField(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Test message",
		zap.Bool("enabled", true),
		zap.Bool("disabled", false))

	output := buf.String()
	assert.Contains(t, output, "enabled=true")
	assert.Contains(t, output, "disabled=false")
}

func TestPipeEncoder_ErrorField(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	err := errors.New("connection timeout")
	logger.Error("Failed to connect", zap.Error(err))

	output := buf.String()
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "Failed to connect")
	assert.Contains(t, output, `error="connection timeout"`)
}

func TestPipeEncoder_Float64Field(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Test message", zap.Float64("daysUntilExpiration", 15.5))

	output := buf.String()
	// Float should be formatted without quotes and show correct value
	assert.Contains(t, output, "daysUntilExpiration=15.50")
	// Verify it does NOT contain the bug value (raw integer representation)
	assert.NotContains(t, output, "4644421991715836928")
}

func TestPipeEncoder_MixedFields(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Token status",
		zap.String("cluster", "production"),
		zap.Int("daysUntilExpiration", 15),
		zap.Bool("needsRefresh", true))

	output := buf.String()
	assert.Contains(t, output, `cluster="production"`)
	assert.Contains(t, output, "daysUntilExpiration=15")
	assert.Contains(t, output, "needsRefresh=true")

	// Verify pipe delimiter between fields
	parts := strings.Split(output, " | ")
	assert.GreaterOrEqual(t, len(parts), 4) // timestamp | level | message | fields...
}

func TestPipeEncoder_NoFields(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Simple message")

	output := buf.String()
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "Simple message")
	// Should end with message, no trailing pipe
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := lines[len(lines)-1]
	assert.True(t, strings.HasSuffix(lastLine, "Simple message"),
		"Expected line to end with message, got: %s", lastLine)
}

func TestPipeEncoder_DurationField(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Operation completed", zap.Duration("elapsed", 5*time.Second))

	output := buf.String()
	assert.Contains(t, output, "elapsed=")
}

func TestPipeEncoder_Clone(t *testing.T) {
	encoder := NewPipeEncoder(" | ")
	clone := encoder.Clone()

	assert.NotNil(t, clone)
	assert.IsType(t, &PipeEncoder{}, clone)
}

func TestPipeEncoder_FullLogFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("Filtering clusters based on --cluster flag",
		zap.Int("matched", 1),
		zap.Int("total", 19))

	output := buf.String()

	// Verify the expected format:
	// timestamp | INFO | Filtering clusters based on --cluster flag | matched=1 | total=19
	assert.Contains(t, output, " | INFO | ")
	assert.Contains(t, output, "Filtering clusters based on --cluster flag")
	assert.Contains(t, output, " | matched=1")
	assert.Contains(t, output, " | total=19")

	// Verify it does NOT contain JSON format
	assert.NotContains(t, output, `{"matched"`)
	assert.NotContains(t, output, `"total":`)
}

func TestPipeEncoder_DryRunPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := createTestLogger(&buf)

	logger.Info("[DRY-RUN] Would regenerate token",
		zap.String("cluster", "production"),
		zap.String("reason", "expires_soon"))

	output := buf.String()
	assert.Contains(t, output, "[DRY-RUN] Would regenerate token")
	assert.Contains(t, output, `cluster="production"`)
	assert.Contains(t, output, `reason="expires_soon"`)
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)
}

func TestNewLoggerWithLevel(t *testing.T) {
	logger := NewLoggerWithLevel(zapcore.DebugLevel)
	assert.NotNil(t, logger)
}
