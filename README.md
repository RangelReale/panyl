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
    "encoding/json"
    "fmt"
    "github.com/RangelReale/panyl"
    "github.com/RangelReale/panyl/plugins/clean"
    "github.com/RangelReale/panyl/plugins/metadata"
    "github.com/RangelReale/panyl/plugins/parse"
    "github.com/RangelReale/panyl/plugins/structure"
    "os"
    "time"
)

func main() {
    processor := panyl.NewProcessor(
        panyl.WithLineLimit(0, 100),
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

    err := processor.Process(os.Stdin, &Output{})
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error processing input: %s", err.Error())
    }
}

type Output struct {
}

func (o *Output) OnResult(p *panyl.Process) (cont bool) {
    var out bytes.Buffer

    // timestamp
    if ts, ok := p.Metadata[panyl.Metadata_Timestamp]; ok {
        out.WriteString(fmt.Sprintf("%s ", ts.(time.Time).Local().Format("2006-01-02 15:04:05.000")))
    }

    // level
    if level := p.Metadata.StringValue(panyl.Metadata_Level); level != "" {
        out.WriteString(fmt.Sprintf("[%s] ", level))
    }

    // category
    if category := p.Metadata.StringValue(panyl.Metadata_Category); category != "" {
        out.WriteString(fmt.Sprintf("{{%s}} ", category))
    }

    // message
    if msg := p.Metadata.StringValue(panyl.Metadata_Message); msg != "" {
        out.WriteString(msg)
    } else if len(p.Data) > 0 {
        // Extracted structure but no metadata
        dt, err := json.Marshal(p.Data)
        if err != nil {
            fmt.Println("Error marshaling data to json: %s", err.Error())
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
    Clean(result *Process) (bool, error)
}
```

### Metadata

```go
// PluginMetadata allows extracting metadata from a line.
// Set result.Metadata with the detected data.
// You can also change result.Line if you need to remove the metadata from the line.
type PluginMetadata interface {
    ExtractMetadata(result *Process) (bool, error)
}
```

### Structure

```go
// PluginStructure allows extracting structure from a line, for example, JSON or XML.
// The full text must be a complete structure, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginStructure interface {
    ExtractStructure(lines ProcessLines, result *Process) (bool, error)
}
```

### Parse

```go
// PluginParse allows parsing data from a line, for example, an Apache log format, a Ruby log format, etc.
// The full text must be completely parsed, partial match should not be supported.
// You should take in account the lines Metdatada/Data and apply them to result at your convenience.
type PluginParse interface {
    ExtractParse(lines ProcessLines, result *Process) (bool, error)
}
```

### Sequence

```go
// PluginSequence allows checking if 2 processes breaks a sequence, for example, if they belong to different
// applications, given it is possible to detect this.
type PluginSequence interface {
    BlockSequence(lastp, p *Process) bool
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
    Consolidate(lines ProcessLines, result *Process) (_ bool, topLines int, _ error)
}
```

### Post Process

```go
// PluginPostProcess is called right before the data is returned to the user, so it allows to do final post-processing
// like detecting some format from a raw structure (JSON or XML), for example, detecting the Apache log format from a
// JSON string.
type PluginPostProcess interface {
    PostProcess(result *Process) (bool, error)
}
```

## Author

Rangel Reale (rangelreale@gmail.com)
