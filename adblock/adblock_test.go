package adblock

import (
	"testing"
)

const STEVENBLACK_BLACKLIST = "https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/fakenews-gambling-porn/hosts"

func TestNewAdblock(t *testing.T) {
	adblocker := NewAdblock(STEVENBLACK_BLACKLIST, nil)

	if adblocker == nil {
		t.Error("Expected NewAdblock to return a valid object")
	}

	if _, found := adblocker.Entries.Get([]byte("ad-assets.futurecdn.net")); !found {
		t.Errorf("Expected to find this entry")
	}

	if _, found := adblocker.Entries.Get([]byte("google.com")); found {
		t.Errorf("Expected to not find this entry")
	}
}
