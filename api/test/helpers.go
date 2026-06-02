package test

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"
)

func LogArtifactFile(t *testing.T) io.Writer {
	dir := t.ArtifactDir()
	filename := fmt.Sprintf("%s_%s_output.txt", time.Now().Format("20060102150405"), t.Name())
	file, err := os.Create(path.Join(dir, filename))
	if err != nil {
		t.Logf("Impossible to write artifacts %v", err)
	}
	return file
}
