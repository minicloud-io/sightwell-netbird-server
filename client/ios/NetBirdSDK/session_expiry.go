package NetBirdSDK

import "time"

// sessionExpiresAtUnix converts the status recorder's session deadline to
// Unix seconds for the gomobile API, where 0 means "no deadline". The zero
// time (no deadline tracked) and instants at or before the Unix epoch map
// to 0.
//
// This file has no build constraint so the conversion can be unit-tested on
// the host; the rest of the package builds for iOS only.
func sessionExpiresAtUnix(deadline time.Time) int64 {
	if deadline.IsZero() {
		return 0
	}
	if secs := deadline.Unix(); secs > 0 {
		return secs
	}
	return 0
}
