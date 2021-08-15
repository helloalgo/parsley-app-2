package child

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"syscall"
)

func signalWatcher(kill <-chan bool, stop <-chan bool, cmd *exec.Cmd) {
	for {
		select {
		case <-kill:
			cmd.Process.Signal(syscall.SIGTERM)
		case <-stop:
			break
		}
	}
}

func runThroughContainer(
	policy ExecutionPolicy,
	limit ExecutionLimit,
	workDir string,
	inputFile string,
	killChan <-chan bool,
	args ExecutionArgs,
) (runResult ExecutionResult) {
	wrappedCommand := "/parsley/bin/core"
	wrappedArgs := []string{
		"-c", fmt.Sprint(limit.RealTime),
		"-r", fmt.Sprint(limit.RealTime),
		"-p", fmt.Sprint(limit.ProcessCount),
		"-s", fmt.Sprint(limit.Memory),
		"-m", fmt.Sprint(limit.Memory),
		"-w", fmt.Sprint(limit.FileWrite),
		"-I", inputFile,
		"--seccomp", limit.Seccomp,
		"--", args.Command,
	}
	wrappedArgs = append(wrappedArgs, args.Args...)

	cmd := exec.Command(wrappedCommand, wrappedArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = workDir
	runResult = ExecutionResult{}

	// Connect empty pipe if stdin is nil
	if policy.StdinPipe == nil {
		var tmp *io.WriteCloser = nil
		policy.StdinPipe = &tmp
	} else {
		stdin, err := cmd.StdinPipe()
		if err == nil {
			*policy.StdinPipe = &stdin
		}
	}
	if policy.StdoutPipe != nil {
		log.Println("Connecting stdout pipe")
		cmd.Stdout = *policy.StdoutPipe
	}
	if policy.StderrPipe != nil {
		log.Println("Connecting stderr pipe")
		cmd.Stderr = *policy.StderrPipe
	}
	err := cmd.Start()
	if err != nil {
		runResult.HostHadError = true
		runResult.HostErrorLocation = "cmd.Start"
		runResult.HostError = err
		return
	}
	// 실행 종료되면 여기에 전송; watcher goroutine에서 받고 종료
	done := make(chan bool, 1)
	go signalWatcher(killChan, done, cmd)
	err = cmd.Wait()
	done <- true
	if err != nil {
		serr, ok := err.(*exec.ExitError)
		if ok {
			if serr.ProcessState.ExitCode() == -1 {
				log.Printf("Terminated: %s", serr)
				runResult.HostHadError = true
				runResult.HostErrorLocation = ""
				runResult.HostError = errors.New("TERMINATED")
			}
		} else {
			runResult.HostHadError = true
			runResult.HostErrorLocation = "cmd.Wait"
			runResult.HostError = err
		}
		return
	}
	// Read and log core log
	logOutput, _ := ioutil.ReadFile(path.Join(workDir, "parsley_runner.log"))
	log.Println(string(logOutput))

	b, err := ioutil.ReadFile(path.Join(workDir, "result.json"))
	if err != nil {
		runResult.HostHadError = true
		runResult.HostErrorLocation = "ioutil.ReadFile"
		runResult.HostError = err
		return
	}
	runResult.CoreResult = string(b)
	runResult.HostHadError = false
	runResult.HostErrorLocation = ""
	runResult.HostError = nil
	return
}
