package pythonu

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jptrs93/goutil/cmdu"
	"github.com/jptrs93/goutil/contextu"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Pool struct {
	scripts        embed.FS
	executablePath string
	entryScript    string
	workers        []*PythonWrapper
	nextInd        int
	nextIndMu      sync.Mutex
	tempDir        string
	ctx            context.Context
}

func NewPool(ctx context.Context, scripts embed.FS, executablePath, entryScript string, n int) *Pool {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		panic("unable to create temporary directory")
	}

	rootDir := findRootDir(ctx, scripts)
	err = fs.WalkDir(scripts, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := scripts.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		fullPath := filepath.Join(tempDir, relPath)
		err = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}
		err = os.WriteFile(fullPath, data, 0644)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialise python scripts temporary dir: %v", err))
	}
	p := &Pool{
		scripts:        scripts,
		executablePath: executablePath,
		entryScript:    entryScript,
		workers:        nil,
		tempDir:        tempDir,
		ctx:            ctx,
	}
	for i := 0; i < n; i++ {
		w := NewPythonWrapper(ctx, executablePath, tempDir, entryScript)
		if _, err := w.InitProcess(); err != nil {
			panic(fmt.Sprintf("failed to initialise python process: %v", err))
		}
		p.workers = append(p.workers, w)
	}
	return p
}

func (p *Pool) Close() {
	for _, w := range p.workers {
		w.Close()
	}
	err := os.RemoveAll(p.tempDir)
	if err != nil {
		slog.ErrorContext(p.ctx, fmt.Sprintf("deleting temporary dir %v: %v", p.tempDir, err))
	}
	slog.InfoContext(p.ctx, fmt.Sprintf("deleted temporary dir: %v", p.tempDir))
}

func CallPool[T any](w *Pool, pythonFunctionName string, inputObj any) (T, error) {
	w.nextIndMu.Lock()
	ind := w.nextInd
	w.nextInd++
	w.nextIndMu.Unlock()
	worker := w.workers[ind%len(w.workers)]
	return Call[T](worker, pythonFunctionName, inputObj)
}

type PythonWrapper struct {
	executablePath string
	scriptPath     string
	executableDir  string
	ctx            context.Context
	cancelCause    context.CancelCauseFunc
	Com            cmdu.PipeCommunication
	cmd            *exec.Cmd
	mu             sync.Mutex
	parentCtx      context.Context
}

func NewPythonWrapper(ctx context.Context, executablePath, workingDir, scriptPath string) *PythonWrapper {
	w := &PythonWrapper{
		executablePath: executablePath,
		scriptPath:     scriptPath,
		executableDir:  workingDir,
		mu:             sync.Mutex{},
		parentCtx:      ctx,
	}

	return w
}

func (w *PythonWrapper) InitProcess() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cmd != nil {
		if w.ctx.Err() != nil {
			// todo log reason from ctx
			slog.WarnContext(w.parentCtx, fmt.Sprintf("python worker process dead (%v), restarting", context.Cause(w.ctx)))
		} else {
			// existing process alive
			return 0, nil
		}
	}

	com, err := cmdu.NewPipeCommunication()
	if err != nil {
		return 0, fmt.Errorf("failed initialising process communication pipe: %w", err)
	}

	ctx, cancelCauseFunc := context.WithCancelCause(w.parentCtx)

	cmd := exec.Command(w.executablePath, w.scriptPath)
	cmd.Dir = w.executableDir
	cmd.Env = os.Environ()
	slog.DebugContext(ctx, fmt.Sprintf("start worker process: working dir: %v, executable: %v, script: %v", cmd.Dir, w.executablePath, w.scriptPath))
	cmd.ExtraFiles = []*os.File{com.OtherRead, com.OtherWrite}

	stdout, stderr, _, closeFunc, err := cmdu.InitStdPipes(cmd)
	if err != nil {
		cancelCauseFunc(fmt.Errorf("failed initialising fitter process stdout/stderr: %w", err))
		return 0, context.Cause(ctx)
	}
	contextu.OnCancel(ctx, closeFunc)

	go consumeStdout(ctx, stdout)
	go consumeStderr(ctx, stderr)
	if err := cmd.Start(); err != nil {
		cancelCauseFunc(fmt.Errorf("failed to start python process: %w", err))
		return 0, context.Cause(ctx)
	}

	contextu.OnCancel(ctx, func() { _ = cmd.Process.Kill() })

	// handle child process exiting
	go func() {
		err := cmd.Wait()
		if err != nil {
			cancelCauseFunc(fmt.Errorf("exit error from python process: %v", err))
			return
		}
		if cmd.ProcessState.ExitCode() != 0 {
			cancelCauseFunc(fmt.Errorf("python process ended with bad exit code %v", cmd.ProcessState.ExitCode()))
			return
		}
		cancelCauseFunc(nil)
	}()

	slog.InfoContext(ctx, "waiting for python script ready signal")

	buf := []byte("ready")
	n, err := com.ThisRead.Read(buf)
	if err != nil {
		cancelCauseFunc(fmt.Errorf("failed to read 'ready' signal from python script: %w", err))
		return 0, context.Cause(ctx)
	} else if n != len(buf) {
		cancelCauseFunc(fmt.Errorf("failed to read 'ready' signal, expected %v bytes but could only read %v", len(buf), n))
		return 0, context.Cause(ctx)
	}

	slog.InfoContext(ctx, "successfully initialised python process")
	w.ctx = ctx
	w.cancelCause = cancelCauseFunc
	w.Com = com
	w.cmd = cmd
	return cmd.Process.Pid, nil
}

func (w *PythonWrapper) Close() {
	w.cancelCause(nil)
}

func Call[T any](w *PythonWrapper, pythonFunctionName string, inputObj any) (T, error) {
	var result T
	if _, err := w.InitProcess(); err != nil {
		return result, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	inputDataBytes, err := json.Marshal(inputObj)
	if err != nil {
		return result, fmt.Errorf("couldn't serialse input data %v", err)
	}
	if err = cmdu.WriteData([]byte(pythonFunctionName), w.Com.ThisWrite); err != nil {
		w.cancelCause(fmt.Errorf("failed writing data to child process: %v", err))
		return result, err
	}
	if err = cmdu.WriteData(inputDataBytes, w.Com.ThisWrite); err != nil {
		w.cancelCause(fmt.Errorf("failed writing data to child process: %v", err))
		return result, err
	}

	var resultData []byte
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	go func() {
		resultData, err = cmdu.ReadData(w.Com.ThisRead)
		cancel()
	}()

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("python Call timed out: %w", ctx.Err())
		w.cancelCause(err)
		return result, err
	}

	if err != nil {
		w.cancelCause(fmt.Errorf("failed reading data to child process: %v", err))
		return result, err
	}
	err = json.Unmarshal(resultData, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshalling result from child process: %v", err)
	}
	return result, nil
}

// findRootDir identifies the first directory of the embedded files
func findRootDir(ctx context.Context, efs embed.FS) string {
	root := "."
	err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." {
			root = path
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("resolving root dir: %v", err))
	}
	return root
}

func consumeStdout(ctx context.Context, stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		slog.InfoContext(ctx, fmt.Sprintf("child process stdout line: %v", scanner.Text()))
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		slog.DebugContext(ctx, fmt.Sprintf("error consuming stdout: %v", err))
	}
}

func consumeStderr(ctx context.Context, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		slog.InfoContext(ctx, fmt.Sprintf("child process stderr line: %v", scanner.Text()))
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		slog.DebugContext(ctx, fmt.Sprintf("error consuming stderr: %v", err))
	}
}
