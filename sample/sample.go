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
