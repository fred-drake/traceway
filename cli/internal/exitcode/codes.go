// Package exitcode defines the stable exit codes emitted by the traceway CLI.
// LLMs and scripts may branch on these values; do not renumber.
package exitcode

const (
	Success     = 0
	Generic     = 1
	Usage       = 2
	Connection  = 3
	Auth        = 4
	NotFound    = 5
	RateLimited = 6
	Server      = 7
)
