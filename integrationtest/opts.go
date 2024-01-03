package integrationtest

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

// Opt is option type for IntegrationTestRunner.
type Opt func(*IntegrationTestRunner) error

// OptBase sets execution base path IntegrationTestRunner.
// OptBase should be usually the first option when passing options to NewIntegrationTestRunner.
func OptBase(base string) Opt {
	return func(itr *IntegrationTestRunner) error {
		absBase, err := filepath.Abs(base)
		if err != nil {
			return fmt.Errorf("getting absolute path for base: '%s' failed: %w", base, err)
		}

		itr.base = absBase
		itr.binHandler.base = absBase
		return nil
	}
}

// OptTarget sets path to compilation target.
func OptTarget(target string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.target = target
		return nil
	}
}

// OptOutput sets output for compilation target.
func OptOutput(output string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.bin = output
		return nil
	}
}

// OptRunArgs adds args to run arguments for test binary.
func OptRunArgs(args ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.runArgs = append(itr.binHandler.runArgs, args...)
		return nil
	}
}

// OptBuildArgs adds args to build arguments for test binary.
func OptBuildArgs(args ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.buildArgs = append(itr.binHandler.buildArgs, args...)
		return nil
	}
}

// OptRunEnv adds env to test binary's run env.
func OptRunEnv(env ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.runEnv = append(itr.binHandler.runEnv, env...)
		return nil
	}
}

// OptBuildEnv adds env to test binary's build env.
func OptBuildEnv(env ...string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.buildEnv = append(itr.binHandler.buildEnv, env...)
		return nil
	}
}

// OptCoverDir sets coverage directory for test binary.
func OptCoverDir(coverDir string) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.binHandler.coverDir = coverDir
		return nil
	}
}

// OptTestMain allows wrapping testing.M into IntegrationTestRunner.
// Example TestMain:
//
//	func TestMain(m *testing.M) {
//		itr := it.NewIntegrationTestRunner(
//			it.OptBase("../"),
//			it.OptTarget("./cmd/app"),
//			it.OptCompose("docker-compose.yaml"),
//			it.OptWaitHTTPReady("http://127.0.0.1:8080", time.Second*10),
//			it.OptTestMain(m),
//		)
//		if err := itr.InitAndRun(); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Before using this pattern be sure to read how TestMain should be used!
func OptTestMain(m *testing.M) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.testRunner = func() error {
			if code := m.Run(); code != 0 {
				return errors.New("tests have failed")
			}
			return nil
		}
		return nil
	}
}

// OptTestFunc allows wrapping testing.T into IntegrationTestRunner.
// Example TestApp:
//
//	func TestApp(t *testing.T) {
//		itr := it.NewIntegrationTestRunner(
//			it.OptBase("../"),
//			it.OptTarget("./cmd/app"),
//			it.OptCompose("docker-compose.yaml"),
//			it.OptWaitHTTPReady("http://127.0.0.1:8080", time.Second*10),
//			it.OptTestFunc(t, testApp),
//		)
//		if err := itr.InitAndRun(); err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	func testApp(t *testing.T) {
//		// run tests here
//	}
//
// This pattern allows setting the env for each test separately.
func OptTestFunc(t *testing.T, fn func(*testing.T)) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.testRunner = func() error {
			fn(t)
			return nil
		}
		return nil
	}
}

// OptCompose adds docker compose stack as pre condition for tests to run.
func OptCompose(composeFile string, opts ...ComposeOpt) Opt {
	return func(itr *IntegrationTestRunner) error {
		compose, err := tc.NewDockerCompose(path.Join(itr.base, composeFile))
		if err != nil {
			return fmt.Errorf("failed to create new compose stack: %w", err)
		}

		c := &composeHandler{c: compose}
		for _, opt := range opts {
			opt(c)
		}

		itr.preHandlers = append(itr.preHandlers, c)
		return nil
	}
}

// OptWaitHTTPReady expects 200 OK from given url before tests can be started.
func OptWaitHTTPReady(url string, timeout time.Duration) Opt {
	return func(itr *IntegrationTestRunner) error {
		itr.ready = func() error {
			started := time.Now()
			for !isReady(url) {
				if time.Since(started) > timeout {
					return fmt.Errorf("readiness deadline %s exceeded", timeout)
				}
				time.Sleep(time.Millisecond * 100)
			}
			return nil
		}
		return nil
	}
}

func isReady(url string) bool {
	r, err := http.Get(url) //nolint:gosec
	if err != nil {
		return false
	}
	defer r.Body.Close()

	return r.StatusCode == http.StatusOK
}