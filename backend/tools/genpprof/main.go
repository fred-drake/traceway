package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"os"

	"github.com/google/pprof/profile"
)

func main() {
	out := flag.String("out", "labeled.pprof", "output file path")
	gz := flag.Bool("gzip", false, "gzip the output (sets the body for an /profiles/ingest POST with Content-Encoding: gzip)")
	flag.Parse()

	p := buildProfile()
	if err := p.CheckValid(); err != nil {
		fail(err)
	}

	f, err := os.Create(*out)
	if err != nil {
		fail(err)
	}
	defer f.Close()

	if *gz {
		w := gzip.NewWriter(f)
		if err := p.Write(w); err != nil {
			fail(err)
		}
		if err := w.Close(); err != nil {
			fail(err)
		}
	} else if err := p.Write(f); err != nil {
		fail(err)
	}

	fmt.Printf("wrote %s (gzip=%v)\n", *out, *gz)
	fmt.Println("expect after decode: 2 cpu samples on 1 stack -> endpoint /checkout=125, /cart=50; request_id dropped")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "genpprof:", err)
	os.Exit(1)
}

func buildProfile() *profile.Profile {
	mainFn := &profile.Function{ID: 1, Name: "main.main", SystemName: "main.main", Filename: "main.go"}
	workFn := &profile.Function{ID: 2, Name: "main.work", SystemName: "main.work", Filename: "work.go"}
	lMain := &profile.Location{ID: 1, Line: []profile.Line{{Function: mainFn, Line: 10}}}
	lWork := &profile.Location{ID: 2, Line: []profile.Line{{Function: workFn, Line: 20}}}

	sample := func(endpoint string, v int64) *profile.Sample {
		return &profile.Sample{
			Location: []*profile.Location{lWork, lMain},
			Value:    []int64{1, v},
			Label:    map[string][]string{"endpoint": {endpoint}, "request_id": {"req-" + endpoint}},
		}
	}

	return &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "samples", Unit: "count"},
			{Type: "cpu", Unit: "nanoseconds"},
		},
		PeriodType:    &profile.ValueType{Type: "cpu", Unit: "nanoseconds"},
		Period:        10_000_000,
		TimeNanos:     1_700_000_000_000_000_000,
		DurationNanos: 1_000_000_000,
		Sample:        []*profile.Sample{sample("/checkout", 100), sample("/cart", 50), sample("/checkout", 25)},
		Function:      []*profile.Function{mainFn, workFn},
		Location:      []*profile.Location{lMain, lWork},
	}
}
