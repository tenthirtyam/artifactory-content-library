// SPDX-License-Identifier: MIT

package vcsp

import (
	"strings"
	"testing"
)

func TestTruncateName(t *testing.T) {
	if truncateName("short") != "short" {
		t.Fatal("short")
	}
	long := strings.Repeat("a", 100) + ".ovf"
	got := truncateName(long)
	if len(got) != maxNameLength || !strings.HasSuffix(got, ".ovf") {
		t.Fatalf("%q len=%d", got, len(got))
	}
	got2 := truncateName(strings.Repeat("b", 100))
	if len(got2) != maxNameLength {
		t.Fatal(got2)
	}
}
