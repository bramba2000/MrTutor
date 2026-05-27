package e2e

import (
	"flag"
	"fmt"
	"io"
	"mrtutor-api/config"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

const (
	execName = "testapp"
)

var e2e = flag.Bool("e2e", false, "run e2e tests")

var host = "http://" + net.JoinHostPort("127.0.0.1", config.Port)

func TestMain(m *testing.M) {
	flag.Parse()
	if e2eDisabled() {
		fmt.Println("SKIP\te2e tests are disabled. Use -e2e flag to enable them.")
		os.Exit(0)
	}
	buildApplication()
	exitCode := m.Run()
	cleanupApplication()
	os.Exit(exitCode)
}

func e2eDisabled() bool {
	return !*e2e
}

func buildApplication() {
	buildCmd := exec.Command("go", "build", "-o", execName, "../..")
	if output, err := buildCmd.Output(); err != nil {
		fmt.Printf("SETUP\tfailed to build application: %v\n\t%s", err, output)
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("build stderr: %s\n", string(exitErr.Stderr))
		}
	}
}

func cleanupApplication() {
	if err := os.Remove(execName); err != nil {
		fmt.Printf("SETUP\tfailed to remove application executable: %v\n", err)
	}
}

func startApplication(t testing.TB) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("./" + execName)
	dir := t.ArtifactDir()
	filename := fmt.Sprintf("%s_%s_output.txt", time.Now().Format("20060102150405"), t.Name())

	file, err := os.Create(path.Join(dir, filename))
	if err != nil {
		t.Logf("Impossible to write artifacts %v", err)
	}
	cmd.Stdout = file

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start application: %v\n", err)
	}
	time.Sleep(1 * time.Second)
	t.Cleanup(func() {
		cmd.Process.Kill()
	})
	return cmd
}

func must[T any](res T, err error) T {
	if err != nil {
		panic(err)
	}
	return res
}

func makeRequest(t testing.TB, method, path string, body io.Reader) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, host+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return http.DefaultClient.Do(req)
}
