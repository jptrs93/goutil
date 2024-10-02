package cmdutil

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type PipeCommunication struct {
	ThisRead   *os.File
	ThisWrite  *os.File
	OtherRead  *os.File
	OtherWrite *os.File
}

func NewPipeCommunication() (PipeCommunication, error) {
	otherRead, thisWrite, err := os.Pipe()
	if err != nil {
		return PipeCommunication{}, fmt.Errorf("creating otherRead/thisWrite pipe: %v", err)
	}
	thisRead, otherWrite, err := os.Pipe()
	if err != nil {
		if e := otherRead.Close(); e != nil {
			slog.Warn(fmt.Sprintf("when closing otherRead for clean up within failed NewPipeCommunication: %v", e))
		}
		if e := thisWrite.Close(); e != nil {
			slog.Warn(fmt.Sprintf("when closing otherRead for clean up within failed NewPipeCommunication: %v", e))
		}
		return PipeCommunication{}, fmt.Errorf("creating thisRead/otherWrite pipe: %v", err)
	}
	pc := PipeCommunication{
		thisRead, thisWrite, otherRead, otherWrite,
	}
	return pc, nil
}

func (pc *PipeCommunication) CloseAndSwallowErrors() {
	_ = pc.Close()
}

func (pc *PipeCommunication) CloseAndLogErrors() {
	errs := pc.Close()
	if len(errs) > 0 {
		slog.Debug(fmt.Sprintf("errors closing PipeCommunication pipes: %q", errs))
	}
}

func (pc *PipeCommunication) Close() []error {
	errs := make([]error, 0, 4)
	if err := pc.ThisWrite.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := pc.OtherWrite.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := pc.ThisRead.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := pc.OtherRead.Close(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func intToBytes(n int) []byte {
	byteSlice := make([]byte, 4)
	binary.BigEndian.PutUint32(byteSlice, uint32(int32(n)))
	return byteSlice
}

func bytesToInt(b []byte) int {
	return int(binary.BigEndian.Uint32(b))
}

func ReadData(thisRead *os.File) ([]byte, error) {
	n := 4
	buf := make([]byte, n) // Buffer to hold the incoming data
	bytesRead, err := thisRead.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if bytesRead != n {
		return nil, fmt.Errorf("not enough bytes read (%v), wanted %v", bytesRead, n)
	}

	n = bytesToInt(buf)

	buf = make([]byte, n) // Buffer to hold the incoming data
	bytesRead, err = thisRead.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if bytesRead != n {
		return nil, fmt.Errorf("not enough bytes read (%v), wanted %v", bytesRead, n)
	}
	return buf, err
}

func WriteData(data []byte, goWrite *os.File) error {

	_, err := goWrite.Write(intToBytes(len(data)))
	if err != nil {
		return err
	}
	_, err = goWrite.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func WriteDataEndSignal(goWrite *os.File) error {
	_, err := goWrite.Write(intToBytes(4294967295))
	if err != nil {
		return err
	}
	return nil
}
