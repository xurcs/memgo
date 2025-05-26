package memory

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
)

var cleanMutex sync.Mutex

func CleanRam() error {
	cleanMutex.Lock()
	defer cleanMutex.Unlock()

	runtime.GC()

	debug.FreeOSMemory()

	switch runtime.GOOS {
	case "windows":
		return cleanRamWindows()
	case "linux", "darwin":
		return nil
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func cleanRamWindows() error {

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setProcessWorkingSetSizeProc := kernel32.NewProc("SetProcessWorkingSetSize")
	emptyWorkingSetProc := kernel32.NewProc("K32EmptyWorkingSet")

	currentProcess, err := syscall.GetCurrentProcess()
	if err != nil {
		return fmt.Errorf("failed to get current process handle: %w", err)
	}

	ret, _, _ := emptyWorkingSetProc.Call(uintptr(currentProcess))

	if ret == 0 {
		minSize := uintptr(0)
		maxSize := uintptr(0)
		ret, _, _ = setProcessWorkingSetSizeProc.Call(
			uintptr(currentProcess),
			minSize,
			maxSize,
		)

		if ret == 0 {
			ret, _, _ = setProcessWorkingSetSizeProc.Call(
				uintptr(currentProcess),
				^uintptr(0), // -1 in two's complement
				^uintptr(0), // -1 in two's complement
			)
		}
	}

	runtime.GC()
	debug.FreeOSMemory()

	tryTriggerMemoryReclaim()

	return nil
}

func tryTriggerMemoryReclaim() {
	defer func() {
		recover()
	}()

	const memoryChunkSize = 100 * 1024 * 1024
	chunk := make([]byte, memoryChunkSize)

	for i := 0; i < len(chunk); i += 4096 {
		chunk[i] = 1
	}

	chunk = nil

	runtime.GC()
	debug.FreeOSMemory()
}
