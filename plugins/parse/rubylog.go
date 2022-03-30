package parse

import (
	"github.com/RangelReale/panyl"
	"regexp"
	"time"
)

var _ panyl.PluginParse = (*RubyLog)(nil)

const RubyLog_Format = "ruby_log"

// RubyLog parses Ruby log lines format
type RubyLog struct {
}

// example: "I, [1999-03-03T02:34:24.895701 #19074]  INFO -- Main: info."

var (
	rubyLogRe           = regexp.MustCompile(`(\w), \[(\d{4}-\d{2}-\d{2}T[^\s]+)\s+(#\d+)]\s+(\w+)\s+--\s+([^:]*):\s+(.*)`)
	rubyTimestampFormat = "2006-01-02T15:04:05.999999999"
)

func (m *RubyLog) ExtractParse(lines panyl.ProcessLines, result *panyl.Process) (bool, error) {
	// Only single line is supported
	if len(lines) != 1 {
		return false, nil
	}

	matches := rubyLogRe.FindStringSubmatch(lines.Line())
	if matches == nil {
		return false, nil
	}

	err := result.MergeLinesData(lines)
	if err != nil {
		return false, err
	}
	result.Line = ""

	timestamp := matches[2]
	level := matches[4]
	message := matches[6]

	result.Data["severity_id"] = matches[1]
	result.Data["timestamp"] = timestamp
	result.Data["pid"] = matches[3]
	result.Data["severity"] = level
	result.Data["prog_name"] = matches[5]
	result.Data["message"] = message

	result.Metadata[panyl.Metadata_Format] = RubyLog_Format
	result.Metadata[panyl.Metadata_Message] = message

	if timestamp != "" {
		ts, err := time.Parse(rubyTimestampFormat, timestamp)
		if err == nil {
			result.Metadata[panyl.Metadata_Timestamp] = ts
		}
	}

	// https://ruby-doc.org/stdlib-2.6.4/libdoc/logger/rdoc/Logger.html
	if level == "DEBUG" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_DEBUG
	} else if level == "INFO" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_INFO
	} else if level == "WARN" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_WARNING
	} else if level == "ERROR" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_ERROR
	} else if level == "FATAL" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_FATAL
	}

	return true, nil
}

func (m RubyLog) IsPanylPlugin() {}
