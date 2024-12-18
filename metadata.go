package panyl

const (
	MetadataStructure           = "structure"
	MetadataFormat              = "format"
	MetadataLevel               = "level"
	MetadataTimestamp           = "ts"      // time.Time
	MetadataTimestampCalculated = "ts_calc" // bool [whether the timestamp was calculated instead of present in the data]
	MetadataMessage             = "message"
	MetadataApplication         = "application"
	MetadataApplicationSource   = "application_source"
	MetadataClean               = "clean" // []string
	MetadataCategory            = "category"
	MetadataOriginalCategory    = "original_category" // [if a plugin changed a category, it can store the original here]
	MetadataCreated             = "created"           // bool [whether the process was created instead of being in the log file]
	MetadataSkip                = "skip"              // bool [if true, the line will be skipped]
)

const (
	MetadataStructureJSON = "json"
	MetadataStructureXML  = "xml"
)

const (
	MetadataLevelTRACE   = "trace"
	MetadataLevelDEBUG   = "debug"
	MetadataLevelINFO    = "info"
	MetadataLevelWARNING = "warn"
	MetadataLevelERROR   = "error"
)

const (
	MetadataCleanAnsiEscape = "ansi_escape"
)
