package blob

import (
	"strings"
	"testing"
	"time"
)

func TestBlobDir(t *testing.T) {
	blob := New()
	dir := blob.dir()
	exp := time.Now().Format("2006/01/06/") + blob.ID.String()
	if !strings.HasSuffix(dir, exp) {
		t.Fatal("expected state dir to be like:" + exp + "\ngot:" + dir)
	}
}
