package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"parsley-app/child"
	pb "parsley-app/proto"
)

// ExecutionContainerSrv is a implemented gRPC server.
type ExecutionContainerSrv struct {
	pb.UnimplementedExecutionContainerServer

	container  child.Host
	outputRate uint32
	inReader   *io.WriteCloser
	outWriter  *bufferedPipe
	errWriter  *bufferedPipe
}

// Create returns a gRPC server instance, initializing the ExecutionHost container.
func Create() (*ExecutionContainerSrv, error) {
	server := ExecutionContainerSrv{}

	container, err := child.InitHost()
	if err != nil {
		return nil, err
	}
	server.container = container

	return &server, nil
}

// SetRunConfig handler
func (s *ExecutionContainerSrv) SetRunConfig(ctx context.Context, req *pb.RunOptions) (*pb.SettingChangeResult, error) {
	log.Println("req: SetRunConfig")
	s.outputRate = req.OutputThrottle
	err := s.container.SetLimits(child.ExecutionLimit{
		RealTime:     req.Limits.RealTime,
		ProcessCount: req.Limits.ProcessCount,
		FileWrite:    req.Limits.Write,
		Memory:       req.Limits.Memory,
		Seccomp:      req.Limits.Seccomp,
		Stream:       req.Limits.StreamSize,
	})
	if err != nil {
		return &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	err = s.container.SetArgs(child.ExecutionArgs{
		Command: req.Params.Command,
		Args:    req.Params.Args,
	})
	if err != nil {
		return &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.SettingChangeResult{
		Success: true,
	}, nil
}

// SetFile handler
// Runner 반환 가능 오류: `STATE_INVALID`, `FILENAME_INVALID`
func (s *ExecutionContainerSrv) SetFile(ctx context.Context, req *pb.SetFileArgs) (*pb.SettingChangeResult, error) {
	log.Printf("req: SetFile")
	err := s.container.SetFile(req.FileName, req.GetData())
	if err != nil {
		return &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &pb.SettingChangeResult{Success: true}, nil
}

// Reset handler
// Runner 반환 가능 오류: `STATE_INVALID`
func (s *ExecutionContainerSrv) Reset(ctx context.Context, req *pb.Empty) (*pb.SettingChangeResult, error) {
	// Clean the old container
	log.Println("req: Reset")
	err := s.container.Clean()
	if err != nil {
		return &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	// Replace with new container
	container, err := child.InitHost()
	if err != nil {
		return &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	s.container = container

	return &pb.SettingChangeResult{Success: true}, err
}

// RunInteractive handler
// Runner 반환 가능 오류: `STATE_INVALID`, `TERMINATED`
func (s *ExecutionContainerSrv) RunInteractive(srv pb.ExecutionContainer_RunInteractiveServer) error {
	log.Printf("req: Start RunInteractive")
	resultChan := make(chan child.ExecutionResult)
	s.inReader = nil

	streamLimitSignal := make(chan bool, 100)

	outStream := buildStreamOutput(s, &srv, pb.OutputType_OT_STDOUT, streamLimitSignal)
	errStream := buildStreamOutput(s, &srv, pb.OutputType_OT_STDERR, streamLimitSignal)
	outStream.WriteLimit = (int64)(s.container.GetLimits().Stream)
	errStream.WriteLimit = (int64)(s.container.GetLimits().Stream)
	s.outWriter = BuildBufferedPipe(&outStream, 100)
	s.errWriter = BuildBufferedPipe(&errStream, 100)
	var outPipe io.Writer = s.outWriter
	var errPipe io.Writer = s.errWriter

	go s.container.Start(child.ExecutionPolicy{
		StdoutPipe: &outPipe,
		StderrPipe: &errPipe,
		StdinPipe:  &s.inReader,
	}, resultChan)

	go func() {
		for {
			inp, err := srv.Recv()
			if err != nil {
				break
			}
			// Pause while inReader is nil (exec not complete)
			for s.inReader == nil {
			}
			_, err = io.WriteString(*s.inReader, inp.Input)
		}
	}()

	execDoneSignal := make(chan bool, 100)
	go func() {
		for {
			select {
			case <-streamLimitSignal:
				s.container.Kill(-5)
				break
			case <-execDoneSignal:
				break
			}
		}
	}()

	result := <-resultChan
	log.Printf("%+v", result)

	execDoneSignal <- true
	s.outWriter.Flush()
	s.errWriter.Flush()

	if result.HostHadError {
		errorMsg := result.HostError.Error()
		if result.HostErrorLocation == "" {
			srv.Send(&pb.RunOutput{
				Type: pb.OutputType_OT_EXIT,
				Data: []byte(fmt.Sprintf("{\"app_error\": true, \"error\": \"%s\"}", errorMsg)),
			})
		}
		srv.Send(&pb.RunOutput{
			Type: pb.OutputType_OT_EXIT,
			Data: []byte(fmt.Sprintf("{\"app_error\": true, \"error\": \"%s: %s\"}", result.HostErrorLocation, errorMsg)),
		})
	} else {
		srv.Send(&pb.RunOutput{
			Type: pb.OutputType_OT_EXIT,
			Data: []byte(fmt.Sprintf("{\"app_error\": false, \"result\": %s}", result.CoreResult)),
		})
	}

	return nil
}

// Stop handler
func (s *ExecutionContainerSrv) Stop(ctx context.Context, req *pb.Empty) (result *pb.SettingChangeResult, err error) {
	log.Println("req: Stop")
	err = s.SendStop(0)
	if err != nil {
		result = &pb.SettingChangeResult{
			Success: false,
			Message: err.Error(),
		}
		return
	}
	result = &pb.SettingChangeResult{Success: true}
	return
}

// SendStop kills the child process and closes the input stream. If reason is not 0, it will be sent to the host.
func (s *ExecutionContainerSrv) SendStop(reason int) (err error) {
	log.Printf("Requested SendStop (intent %d)", reason)
	s.outWriter.Close()
	s.errWriter.Close()
	err = s.container.Kill(reason)
	if err != nil {
		return
	}
	err = (*s.inReader).Close()
	return
}
