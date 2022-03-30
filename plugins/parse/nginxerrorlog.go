package parse

import (
	"github.com/RangelReale/panyl"
	"regexp"
	"time"
)

var _ panyl.PluginParse = (*NGINXErrorLog)(nil)

const NGINXErrorLog_Format = "nginx_error_log"

// NGINXErrorLog parses NGINX log lines format
type NGINXErrorLog struct {
}

// 2022/03/10 20:20:48 [error] 62#62: *70 invalid URL prefix in "", client: 127.0.0.1, server: , request: "GET / HTTP/1.1", host: "localhost:8080", referrer: "http://localhost:8080"

var (
	// https://stackoverflow.com/a/26125951/784175
	nginxErrorLogRe           = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(\w+)]\s+(\d+)#(\d+):\s+((\*\w+)\s+)?(.*)$`)
	nginxErrorTimestampFormat = "2006/01/02 15:04:05"
)

func (m *NGINXErrorLog) ExtractParse(lines panyl.ProcessLines, result *panyl.Process) (bool, error) {
	// Only single line is supported
	if len(lines) != 1 {
		return false, nil
	}

	matches := nginxErrorLogRe.FindStringSubmatch(lines.Line())
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
	message := matches[7]

	result.Data["timestamp"] = timestamp
	result.Data["level"] = level
	result.Data["pid"] = matches[3]
	result.Data["tid"] = matches[4]
	result.Data["cid"] = matches[6]
	result.Data["message"] = message

	result.Metadata[panyl.Metadata_Format] = NGINXErrorLog_Format
	result.Metadata[panyl.Metadata_Message] = message

	if timestamp != "" {
		ts, err := time.Parse(nginxErrorTimestampFormat, timestamp)
		if err == nil {
			result.Metadata[panyl.Metadata_Timestamp] = ts
		}
	}

	// https://github.com/phusion/nginx/blob/master/src/core/ngx_log.c#L56
	switch level {
	case "debug":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_DEBUG
	case "info", "notice":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_INFO
	case "warn":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_WARNING
	case "error":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_ERROR
	case "alert", "crit":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_CRITICAL
	case "emerg":
		result.Metadata[panyl.Metadata_Level] = panyl.MetadataLevel_FATAL
	}

	return true, nil
}

func (m NGINXErrorLog) IsPanylPlugin() {}
