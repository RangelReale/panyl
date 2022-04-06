package panyl

import "strconv"

const (
	Metadata_Structure           = "structure"
	Metadata_Format              = "format"
	Metadata_Level               = "level"
	Metadata_Timestamp           = "ts"      // time.Time
	Metadata_TimestampCalculated = "ts_calc" // bool [whether the timestamp was calculated instead of present in the data]
	Metadata_Message             = "message"
	Metadata_Application         = "application"
	Metadata_Clean               = "clean" // []string
	Metadata_Category            = "category"
	Metadata_OriginalCategory    = "original_category" // [if a plugin changed a category, it can store the original here]
	Metadata_Created             = "created"           // bool [whether the process was created instead of being in the log file]
)

const (
	MetadataStructure_JSON = "json"
	MetadataStructure_XML  = "xml"
)

const (
	MetadataLevel_TRACE    = "trace"
	MetadataLevel_DEBUG    = "debug"
	MetadataLevel_INFO     = "info"
	MetadataLevel_WARNING  = "warn"
	MetadataLevel_ERROR    = "error"
	MetadataLevel_CRITICAL = "critical"
	MetadataLevel_FATAL    = "fatal"
)

const (
	MetadataClean_AnsiEscape = "ansi_escape"
)

// MapValue is a helper for handling map[string]interface{}
type MapValue map[string]interface{}

func (m MapValue) HasValue(name string) bool {
	_, ok := m[name]
	return ok
}

func (m MapValue) StringValue(name string) string {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case string:
			return vv
		}
	}
	return ""
}

func (m MapValue) IntValue(name string) int {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case int:
			return vv
		case int8:
			return int(vv)
		case int16:
			return int(vv)
		case int32:
			return int(vv)
		case int64:
			return int(vv)
		case uint:
			return int(vv)
		case uint8:
			return int(vv)
		case uint16:
			return int(vv)
		case uint32:
			return int(vv)
		case uint64:
			return int(vv)
		}
	}
	return 0
}

func (m MapValue) BoolValue(name string) bool {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case bool:
			return vv
		case int:
			return vv != 0
		case string:
			b, err := strconv.ParseBool(vv)
			if err == nil {
				return b
			}
		}
	}
	return false
}

func (m MapValue) ListValue(name string) []string {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case []string:
			return vv
		}
	}
	return nil
}

func (m MapValue) ListValueAdd(name string, value string) {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case []string:
			// check duplicates
			for _, vdup := range vv {
				if vdup == value {
					return
				}
			}
			m[name] = append(vv, value)
			return
		}
	}
	m[name] = []string{value}
}

func (m MapValue) ListValueContains(name string, value string) bool {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case []string:
			for _, v := range vv {
				if v == value {
					return true
				}
			}
		}
	}
	return false
}
