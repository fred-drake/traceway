package output

import (
	"io"
	"text/tabwriter"
)

// NewTabWriter returns a *text/tabwriter.Writer configured for traceway's
// table output style: left-aligned columns separated by two spaces. Callers
// must call Flush() after writing all rows.
//
//	tw := output.NewTabWriter(w)
//	fmt.Fprintln(tw, "ID\tNAME")
//	fmt.Fprintf(tw, "%s\t%s\n", id, name)
//	tw.Flush()
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}
