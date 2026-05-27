package e2e

import (
	"flag"
	"fmt"
	"net"
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

// startApplication starts the app on a free port and returns its base URL and the process handle.
// The process is killed automatically via t.Cleanup, so callers only need cmd when sending signals.
func startApplication(t testing.TB) (host string, cmd *exec.Cmd) {
	t.Helper()

	port := freePort(t)
	host = "http://" + net.JoinHostPort("127.0.0.1", port)

	cmd = exec.Command("./" + execName)
	cmd.Env = append(os.Environ(), "PORT="+port)

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
	t.Cleanup(func() {
		cmd.Process.Kill()
	})
	return host, cmd
}
