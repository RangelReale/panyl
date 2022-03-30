package parse

import (
	"github.com/RangelReale/panyl"
	"regexp"
	"time"
)

var _ panyl.PluginParse = (*GoLog)(nil)

const GoLog_Formag = "go_log"

// GoLog parse Golang log lines format
type GoLog struct {
}

// example: "2022-03-10T19:53:21.434Z	INFO	datadog-go/tracer.go:35	Datadog Tracer v1.28.0 ERROR: lost 2 traces"

var goLogRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T[^\s]+)\s+(\w+)\s+(.*\.go:\d+)\s+(.*)$`)

func (m *GoLog) ExtractParse(lines panyl.ProcessLines, result *panyl.Process) (bool, error) {
	// Only single line is supported
	if len(lines) != 1 {
		return false, nil
	}

	matches := goLogRe.FindStringSubmatch(lines.Line())
	if matches == nil {
		return false, nil
	}

	err := result.MergeLinesData(lines)
	if err != nil {
		return false, err
	}
	result.Line = ""

	timestamp := matches[1]
	level := matches[2]
	message := matches[4]

	result.Data["timestamp"] = timestamp
	result.Data["level"] = level
	result.Data["source"] = matches[3]
	result.Data["message"] = message

	result.Metadata[panyl.Metadata_Format] = GoLog_Formag
	result.Metadata[panyl.Metadata_Message] = message

	if timestamp != "" {
		ts, err := time.Parse(time.RFC3339Nano, timestamp)
		if err == nil {
			result.Metadata[panyl.Metadata_Timestamp] = ts
		}
	}
	if level == "DEBUG" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_DEBUG
	} else if level == "INFO" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_INFO
	} else if level == "WARN" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_WARNING
	} else if level == "ERROR" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_ERROR
	}

	return true, nil
}

func (m GoLog) IsPanylPlugin() {}
