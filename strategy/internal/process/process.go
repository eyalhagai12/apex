package process

import (
	"apex/internal/domain"
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

type managedProcess struct {
	cmd  *exec.Cmd
	done chan error
}

func run(scriptPath string) (*managedProcess, error) {
	cmd := exec.Command("python", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	p := &managedProcess{
		cmd:  cmd,
		done: make(chan error, 1),
	}

	go func() {
		p.done <- cmd.Wait()
	}()

	return p, nil
}

func (mp *managedProcess) stop(timeout time.Duration) error {
	pid := uint32(mp.cmd.Process.Pid)

	if err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, pid); err != nil {
		return mp.forceStop()
	}

	select {
	case err := <-mp.done:
		return err
	case <-time.After(timeout):
		return mp.forceStop()
	}
}

func (mp *managedProcess) forceStop() error {
	kill := exec.Command("taskkill", "/T", "/F", "/PID",
		strconv.Itoa(mp.cmd.Process.Pid))
	kill.Run()
	return <-mp.done
}

type ProcessDeployer struct {
	activeProcesses map[string]*managedProcess
	rw              sync.RWMutex
}

func New() *ProcessDeployer {
	return &ProcessDeployer{
		activeProcesses: make(map[string]*managedProcess, 10),
		rw:              sync.RWMutex{},
	}
}

func (pd *ProcessDeployer) Deploy(ctx context.Context, strategy *domain.Strategy) (string, error) {
	mp, err := run(strategy.Path)
	if err != nil {
		return "", err
	}

	key := strategy.ID.String()
	pd.rw.Lock()
	pd.activeProcesses[key] = mp
	pd.rw.Unlock()

	return key, nil
}

func (pd *ProcessDeployer) Teardown(ctx context.Context, key string) error {
	pd.rw.RLock()
	process, ok := pd.activeProcesses[key]
	if !ok {
		return errors.New("no such process")
	}
	pd.rw.RUnlock()

	if err := process.stop(30 * time.Second); err != nil {
		return err
	}

	delete(pd.activeProcesses, key)

	return nil
}
