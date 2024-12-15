# Panyl

Panyl (Parse ANY Log) is a Golang library to parse logs that may have mixed formats,
like log files for multiple services in the same file.

It parses each line trying to find some format or structure, using a series of plugins to extract metadata,
detect common structure like JSON or XML, parsing line formats like Ruby or MongoDB logs,
and helping handling multi-line logs, checking for structures successively, or asking plugins to find custom data 
in a serie of lines.

As log formats vary widely, and also internal services may have some peculiarities that prevents having a standard
way of extracting metadata, the recommended way of using this library is creating your
own plugins customized for your needs, and using the [panyl-cli](https://github.com/RangelReale/panyl-cli) to create a
customizable cli for your needs.

## Example

This examples parses from stdin any of the formats registered as plugins (like JSON, Ruby logs, MongoDB logs), removing 
any Ansi color formatting from each line, and extracing application name information from docker-compose logs.

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/RangelReale/panyl/v2"
    "github.com/RangelReale/panyl-plugins/v2/parse"
    "github.com/RangelReale/panyl/v2/plugins/clean"
    "github.com/RangelReale/panyl-plugins/v2/metadata"
    "github.com/RangelReale/panyl/v2/plugins/structure"
)

func main() {
    ctx := context.Background()
    processor := panyl.NewProcessor(
        panyl.WithPlugins(
            &clean.AnsiEscape{},
            &metadata.DockerCompose{},
            &structure.JSON{},
            &parse.GoLog{},
            &parse.RubyLog{},
            &parse.MongoLog{},
            &parse.NGINXErrorLog{},
        ),
        // may use a logger when debugging, it outputs each source line and parsed processes
        // panyl.WithLogger(panyl.NewStdLogOutput()),
    )

    err := processor.Process(ctx, os.Stdin, &Output{}, panyl.WithLineLimit(0, 100))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error processing input: %s", err.Error())
    }
}

type Output struct {
}

func (o *Output) OnResult(ctx context.Context, p *panyl.Process) (cont bool) {
    var out bytes.Buffer

    // timestamp
    if ts, ok := p.Metadata[panyl.MetadataTimestamp]; ok {
        out.WriteString(fmt.Sprintf("%s ", ts.(time.Time).Local().Format("2006-01-02 15:04:05.000")))
    }

    // level
    if level := p.Metadata.StringValue(panyl.MetadataLevel); level != "" {
        out.WriteString(fmt.Sprintf("[%s] ", level))
    }

    // category
    if category := p.Metadata.StringValue(panyl.MetadataCategory); category != "" {
        out.WriteString(fmt.Sprintf("{{%s}} ", category))
    }

    // message
    if msg := p.Metadata.StringValue(panyl.MetadataMessage); msg != "" {
        out.WriteString(msg)
    } else if len(p.Data) > 0 {
        // Extracted structure but no metadata
        dt, err := json.Marshal(p.Data)
        if err != nil {
            fmt.Printf("Error marshaling data to json: %s\n", err.Error())
            return
        }
        out.WriteString(fmt.Sprintf("| %s", string(dt)))
    } else if p.Line != "" {
        // Show raw line if available
        out.WriteString(p.Line)
    }

    fmt.Println(out.String())
    return true
}
```

## Plugin types

### Clean

```go
// PluginClean allows cleaning of a line.
// Change result.Line if you need to modify the line.
// You can set result.Metadata to allow other plugins to detect the change.
type PluginClean interface {
    Clean(ctx context.Context, result *Process) (bool, error)
}
```

### Metadata

```go
// PluginMetadata allows extracting metadata from a line.
// Set result.Metadata with the detected data.
// You can also change result.Line if you need to remove the metadata from the line.
type PluginMetadata interface {
    ExtractMetadata(ctx context.Context, result *Process) (bool, error)
}
```

### Structure

```go
// PluginStructure allows extracting structure from a line, for example, JSON or XML.
// The full text must be a complete structure, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginStructure interface {
    ExtractStructure(ctx context.Context, lines ProcessLines, result *Process) (bool, error)
}
```

### Parse

```go
// PluginParse allows parsing data from a line, for example, an Apache log format, a Ruby log format, etc.
// The full text must be completely parsed, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginParse interface {
    ExtractParse(ctx context.Context, lines ProcessLines, result *Process) (bool, error)
}
```

### Sequence

```go
// PluginSequence allows checking if 2 processes breaks a sequence, for example, if they belong to different
// applications, given it is possible to detect this.
type PluginSequence interface {
    BlockSequence(ctx context.Context, lastp, p *Process) bool
}
```

### Consolidate

```go
// PluginConsolidate allows to consolidate lines that couldn't be parsed by any plugin, like for example,
// multi-line Ruby error strings.
// The plugin should ALWAYS read lines from the top of the list, and set data in result about them.
// The topLines result states how many lines were processed, and they will be removed from future calls.
// The plugin can be called multiple times for the same set of lines, so don't try to detect more if you
// find a line that don't match, you will be called again after the unmatched line.
type PluginConsolidate interface {
    Consolidate(ctx context.Context, lines ProcessLines, result *Process) (_ bool, topLines int, _ error)
}
```

### ParseFormat

 ```go
// PluginParseFormat is called for results that don't have Metadata_Format set, so it allows
// detecting some format from a raw structure (JSON or XML), for example, detecting the Apache log format from
// the parsed JSON data.
type PluginParseFormat interface {
    ParseFormat(ctx context.Context, result *Process) (bool, error)
}
```

### Create

```go
// PluginCreate allows creating process entries that are not present in the log file.
// Use this to add custom log entries to the output.
// This is called after PluginPostProcess, and PluginPostProcess is also called for each item.
// Metadata_Created is set as true for items created by these functions.
type PluginCreate interface {
    CreateBefore(ctx context.Context, result *Process) ([]*Process, error)
    CreateAfter(ctx context.Context, result *Process) ([]*Process, error)
}
```

### Post Process

```go
// PluginPostProcess is called right before the data is returned to the user, so it allows to do any final
// post-processing on the data.
// Order determines in which order post process plugins execute, lower execute first than higher.
// Use PostProcessOrder_Default as default. PostProcessOrder_First and PostProcessOrder_Last should be used
// as limits.
type PluginPostProcess interface {
    PostProcessOrder() int
    PostProcess(ctx context.Context, result *Process) (bool, error)
}
```

## Plugin execution order

- line received from source: `process.Line` = line, `process.RawSource` = line
- `PluginClean`: `process.Line` changes to be cleaned, like removing ANSI codes
- `process.Line` is trimmed with `strings.TrimSpace`
- `PluginMetadata`: `process.Metadata` may be changed with extracted metadata (like application names in docker-compose logs), 
  `process.Line` may be changed removing the metadata information.
- `process.Source` is set to the current `process.Line`
- add current line to a list of unprocessed lines to support multiline parsing
- `PluginStructure`: may extract structured data (like JSON) to `process.Metadata` and/or `process.Data` from the list of lines
- `PluginParse`: may detect data and/or metadata from line-based formats (like Apache logs)
- `PluginSequence`: if no known format was found, sequence plugins can check for sequence breaks, like docker-compose logs
  having the application name changed
- `PluginConsolidate`: some logs can output multiple lines, like Ruby logs, or multiline JSON. This plugin can be used
  to detect a format and consolidate from multiple lines
- otherwise, if known data was found:
- `PluginParseFormat`: if `MetadataFormat` metadata was not set, this plugin is called to try to detect a format from the
  available data. This is used to detect formats from general structures, like Apache logs in JSON format.
- `PluginPostProcess`: this plugin can be used to change processed items before they are returned
- if `MetadataTimestamp` was not set, a timestamp is derived from the timestamp of the last sent record, if available
- if `MetadataSkip` is set to true, the record is not sent to the output and is discarded
- `PluginCreate.CreateBefore`: can be used to create items based on the item about to be output, to be returned before it.
- The processed item is returned to `ProcessResult`
- `PluginCreate.CreateAfter`: can be used to create items based on the item about to be output, to be returned after it.

## Author

Rangel Reale (rangelreale@gmail.com)
