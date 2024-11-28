package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new default logger
// it will need to be closed with
// ```
// defer logger.Desugar().Sync()
// ```
// to make sure all has been piped out before terminating
func NewLogger(l string) *zap.SugaredLogger {
	var lvl string

	val := strings.ToLower(l)

	switch val {
	case "debug", "error", "warn", "info":
		lvl = val
	case "warning":
		lvl = "warn"
	default:
		lvl = "error"
	}

	rawJSON := []byte(
		fmt.Sprintf(
			`{
				"level": "%s",
				"encoding": "json",
				"outputPaths": ["stdout"],
				"errorOutputPaths": ["stdout","stderr"],
				"encoderConfig": {
					"messageKey": "message",
					"levelKey": "severity",
					"levelEncoder": "lowercase",
					"timeKey": "@timestamp",
					"timeEncoder": "rfc3339nano"
				}
			}`,
			lvl),
	)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(cfg.EncoderConfig), zapcore.AddSync(os.Stdout), cfg.Level)

	return zap.New(core).Sugar()

}
