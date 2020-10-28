package child

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/rs/xid"
)

// StateInvalid indicates the status of the invoked action is invalid.
var errStateInvalid = errors.New("STATE_INVALID")
var errFilenameInvalid = errors.New("FILENAME_INVALID")
var errCommandEmpty = errors.New("COMMAND_EMPTY")

// HostStatus is used to save and report the status of a ExecutionHost.
const (
	HostRunning = "running"
	HostReady   = "ready"
	HostLoading = "loading"
	HostClosed  = "closed"
)

type Host struct {
	*sync.Mutex
	key        string
	workDir    string
	args       ExecutionArgs
	limits     ExecutionLimit
	status     string
	killChan   chan bool
	killReason int
}

// createWorkDir creates a empty directory for the current ExecutionHost key.
func (host *Host) setupSession() (err error) {
	newKey := xid.New().String()
	newPath := os.Getenv("PARSLEY")
	log.Printf("InitHost: Key %s, WorkDir %s", newKey, newPath)
	if newPath == "" {
		newPath = "/parsley"
	}
	newPath = fmt.Sprintf("%s/tmp/%s", newPath, newKey)
	err = os.MkdirAll(newPath, 0770)
	if err != nil {
		return
	}
	os.Chmod(newPath, 0770)
	host.workDir = newPath
	host.key = newKey
	host.limits = ExecutionLimit{
		RealTime:     1000,
		ProcessCount: 1,
		FileWrite:    0,
		Memory:       128 * 1024,
		Seccomp:      "basic",
	}
	host.status = HostReady
	return
}

// Clean 메소드는 모든 실행을 마치고 작업 폴더를 삭제합니다.
func (host *Host) Clean() (err error) {
	host.Lock()
	if host.status == HostRunning {
		host.Kill(0)
		if err != nil {
			return
		}
	}
	log.Printf("Clean WorkDir %s", host.workDir)
	err = os.RemoveAll(host.workDir)
	host.status = HostClosed
	host.Unlock()
	return
}

// Kill 메소드는 자식 프로세스가 실행중일 경우 SIGTERM 시그널을 보냅니다.
func (host *Host) Kill(reason int) error {
	log.Printf("Host Kill requested: reason %d", reason)
	if host.status != HostRunning {
		return errStateInvalid
	}
	host.killReason = reason
	host.killChan <- true
	return nil
}

// SetFile 메소드는 `name` 파일명으로 파일을 저장합니다. 파일은 폴더 안에 들어갈 수 없으며, ~ 또는 / 가 포함되서는 안됩니다.
func (host *Host) SetFile(name string, data []byte) (err error) {
	host.Lock()
	defer host.Unlock()
	if strings.ContainsAny(name, "~/") {
		err = errFilenameInvalid
		return
	}
	filePath := path.Join(host.workDir, name)
	if !strings.HasPrefix(filePath, host.workDir) {
		err = errFilenameInvalid
		return
	}
	err = ioutil.WriteFile(filePath, data, 0770)
	log.Printf("SetFile %s, %d bytes", filePath, len(data))
	return
}

// SetArgs sets the execution arguments for the current host.
func (host *Host) SetArgs(args ExecutionArgs) (err error) {
	host.Lock()
	defer host.Unlock()
	if host.status != HostReady {
		err = errStateInvalid
	} else {
		host.args = args
	}
	return
}

// SetLimits sets the execution limits for the current host.
func (host *Host) SetLimits(limit ExecutionLimit) (err error) {
	host.Lock()
	defer host.Unlock()
	if host.status != HostReady {
		err = errStateInvalid
	} else {
		host.limits = limit
	}
	return
}

// GetArgs is a getter.
func (host Host) GetArgs() ExecutionArgs {
	return host.args
}

// GetLimits is a getter.
func (host Host) GetLimits() ExecutionLimit {
	return host.limits
}

// GetStatus is a getter.
func (host Host) GetStatus() string {
	return host.status
}

// Start 메소드는 실행을 시작합니다.
func (host *Host) Start(policy ExecutionPolicy, report chan ExecutionResult) {
	host.Lock()
	if host.status != HostReady {
		report <- ExecutionResult{
			HostErrorLocation: "",
			HostHadError:      true,
			HostError:         errStateInvalid,
		}
		host.Unlock()
		return
	}
	host.Unlock()
	if host.args.Command == "" {
		report <- ExecutionResult{
			HostErrorLocation: "",
			HostHadError:      true,
			HostError:         errCommandEmpty,
		}
		return
	}
	host.Lock()
	host.status = HostRunning
	host.killChan = make(chan bool, 100)
	host.Unlock()
	result := runThroughContainer(policy, host.limits, host.workDir, host.killChan, host.args)

	// Check for killReason if child is terminated
	if result.HostHadError && result.HostError.Error() == "TERMINATED" && host.killReason != 0 {
		result.HostError = fmt.Errorf("INTENDED: %d", host.killReason)
	}

	report <- result
	host.killReason = 0
	host.Lock()
	host.status = HostReady
	host.Unlock()
}

// InitHost 메소드는 ExecutionHost을 초기화하고 생성된 객체를 반환합니다.
func InitHost() (host Host, err error) {
	host = Host{}
	host.Mutex = &sync.Mutex{}
	host.args = ExecutionArgs{
		Command: "echo",
		Args:    []string{"Arguments not set"},
	}
	host.setupSession()
	return
}
