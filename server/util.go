package server

import (
	"bufio"
	"io"
	"log"
	pb "parsley-app/proto"
	"time"
)

type streamOutput struct {
	dataType   pb.OutputType
	writeSize  uint64
	WriteLimit int64
	stream     *pb.ExecutionContainer_RunInteractiveServer
	killSignal chan<- bool
	closed     bool
}

func min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

func (s *streamOutput) Write(p []byte) (n int, err error) {
	if s.closed {
		return len(p), nil
	}
	err = (*s.stream).Send(&pb.RunOutput{
		Type: s.dataType,
		Data: p,
	})
	n = len(p)
	s.writeSize += (uint64)(n)
	if s.WriteLimit > 0 && (s.writeSize > (uint64)(s.WriteLimit)) {
		log.Printf("WriteLimit exceeded! Limit %d, Current %d", s.WriteLimit, s.writeSize)
		s.killSignal <- true
		s.closed = true
	}
	return
}

func buildStreamOutput(
	srv *ExecutionContainerSrv,
	stream *pb.ExecutionContainer_RunInteractiveServer,
	dataType pb.OutputType,
	killSignal chan<- bool,
) streamOutput {
	return streamOutput{
		dataType:   dataType,
		writeSize:  0,
		WriteLimit: -1,
		stream:     stream,
		killSignal: killSignal,
		closed:     false,
	}
}

type savingWriter struct {
	data *[]byte
}

func (s savingWriter) Write(p []byte) (int, error) {
	*s.data = append(*s.data, p...)
	return len(p), nil
}

func (s savingWriter) Get() []byte {
	return *s.data
}

type bufferedPipe struct {
	source       *bufio.Writer
	readInterval uint32
	lastFlush    int64
	closed       bool
}

func (b *bufferedPipe) Write(p []byte) (n int, err error) {
	if b.closed {
		return 0, nil
	}
	n, err = b.source.Write(p)
	if err != nil {
		return
	}
	now := time.Now().UnixNano()
	if (now - b.lastFlush) > (int64)(b.readInterval) {
		b.lastFlush = now
		err = b.source.Flush()
	}
	return
}

func (b *bufferedPipe) Flush() (err error) {
	err = b.source.Flush()
	return
}

func (b *bufferedPipe) Close() {
	b.closed = true
}

// BuildBufferedPipe builds a bufferedPipe struct. interval is given by milliseconds.
func BuildBufferedPipe(w io.Writer, interval uint32) (v *bufferedPipe) {
	v = &bufferedPipe{}
	v.readInterval = interval * 1000000
	v.source = bufio.NewWriter(w)
	v.lastFlush = time.Now().UnixNano()
	v.closed = false
	return
}
