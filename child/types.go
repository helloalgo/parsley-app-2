package child

import "io"

// ExecutionLimit 구조체는 child process가 사용할 수 있는 자원의 한계를 정의합니다.
type ExecutionLimit struct {
	// 실행 wall time (ms)
	RealTime uint32
	// 프로세스 수
	ProcessCount uint32
	// 최대 생성 파일 크기 (KB)
	FileWrite uint32
	// 최대 메모리 (KB)
	Memory uint32
	// Seccomp 정책 (코어 문서 참조)
	Seccomp string
	// stdout, stderr 출력 크기 (각각)
	Stream uint32
}

// ExecutionPolicy 구조체는 child process의 실행 정책을 정의합니다.
type ExecutionPolicy struct {
	// nil일 경우 출력을 반환하지 않습니다.
	StdoutPipe *io.Writer
	// nil일 경우 출력을 반환하지 않습니다.
	StderrPipe *io.Writer
	// nil일 경우 입력을 받지 않습니다.
	StdinPipe **io.WriteCloser
}

// ExecutionResult 구조체는 명령어 실행결과를 정의합니다.
type ExecutionResult struct {
	// App 오류 발생위치
	HostErrorLocation string
	// App에서 오류가 발생했는지
	HostHadError bool
	// App에서 발생한 오류
	HostError error
	// 실행 결과 json
	CoreResult string
}

// ExecutionArgs 구조체는 실행 명령을 정의합니다.
type ExecutionArgs struct {
	Command string
	Args    []string
}

// KillChanMessage 구조체는 자식 프로세스에 종료 명령을 보낼 떄 필요한 자료형입니다.
type KillChanMessage struct {
	reason int
}
