package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestShowJSONAppliesTransformOnce(t *testing.T) {
	t.Setenv("FORCE_COLOR", "0")
	in := gjson.Parse(`{"data":{"data":1,"x":2}}`)
	out, err := os.CreateTemp(t.TempDir(), "json")
	if err != nil {
		t.Fatal(err)
	}
	if err := ShowJSON(out, "sample", in, "json", "data"); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.TrimSpace(text) == "1" || !strings.Contains(text, `"x"`) {
		t.Fatalf("transform applied twice, got %s", text)
	}
}

func TestStreamOutput(t *testing.T) {
	t.Setenv("PAGER", "cat")
	err := streamOutput("stream test", func(w *os.File) error {
		_, writeErr := w.WriteString("Hello world\n")
		return writeErr
	})
	if err != nil {
		t.Errorf("streamOutput failed: %v", err)
	}
}
