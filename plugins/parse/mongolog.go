package parse

import (
	"github.com/RangelReale/panyl"
	"regexp"
	"time"
)

var _ panyl.PluginParse = (*MongoLog)(nil)

const MongoLog_Format = "mongo_log"

// MongoLog parses MongoDB log lines format
type MongoLog struct {
}

// example: "2022-03-10T20:17:25.039+0000 I  NETWORK  [conn15] end connection 172.19.0.1:49812 (7 connections now open)"

var (
	mongoLogRe           = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T[^\s]+)\s+(\w)\s+(\w+)\s+\[(\w+)]\s+(.*)$`)
	mongoTimestampFormat = "2006-01-02T15:04:05.999999999-0700"
)

func (m *MongoLog) ExtractParse(lines panyl.ProcessLines, result *panyl.Process) (bool, error) {
	// Only single line is supported
	if len(lines) != 1 {
		return false, nil
	}

	matches := mongoLogRe.FindStringSubmatch(lines.Line())
	if matches == nil {
		return false, nil
	}

	err := result.MergeLinesData(lines)
	if err != nil {
		return false, err
	}
	result.Line = ""

	timestamp := matches[1]
	severity := matches[2]
	message := matches[5]

	result.Data["timestamp"] = timestamp
	result.Data["severity"] = severity
	result.Data["component"] = matches[3]
	result.Data["context"] = matches[4]
	result.Data["message"] = message

	result.Metadata[panyl.Metadata_Format] = MongoLog_Format

	if timestamp != "" {
		ts, err := time.Parse(mongoTimestampFormat, timestamp)
		if err == nil {
			result.Metadata[panyl.Metadata_Timestamp] = ts
		}
	}

	// https://docs.mongodb.com/manual/reference/log-messages/
	if severity == "D" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_DEBUG
	} else if severity == "I" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_INFO
	} else if severity == "w" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_WARNING
	} else if severity == "E" {
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_ERROR
	}

	return true, nil
}

func (m MongoLog) IsPanylPlugin() {}
