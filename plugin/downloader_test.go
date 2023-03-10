package plugin

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloader_Download_whenPluginIsAvailableLocally(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "p-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	arbitraryPluginDistPath := path.Join(tmpDir, fmt.Sprintf("arbitrary-plugin-1.0.0-%s-%s.zip", runtime.GOOS, runtime.GOARCH))
	if err := os.WriteFile(arbitraryPluginDistPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	arbitraryPm, _ := NewPluginManager("arbitraryName", &Settings{
		BaseDir: EnvironmentAwaredValue(tmpDir),
	}, false, false, "")
	testObject := NewDownloader(arbitraryPm)

	actualPath, err := testObject.Download(&PluginDefinition{
		Name:    "arbitrary-plugin",
		Version: "1.0.0",
	})

	assert.NoError(t, err)
	assert.Equal(t, arbitraryPluginDistPath, actualPath)
}
