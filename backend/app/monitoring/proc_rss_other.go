//go:build !linux

package monitoring

func ReadRSSBytes() (uint64, bool) {
	return 0, false
}
