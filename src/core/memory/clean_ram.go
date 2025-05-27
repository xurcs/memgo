package memory

import (
	"fmt"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"unsafe"
)

var cleanMutex sync.Mutex

type SYSTEM_CACHE_INFORMATION struct {
	CurrentSize       uintptr
	PeakSize          uintptr
	PageFaultCount    uint32
	MinimumWorkingSet uintptr
	MaximumWorkingSet uintptr
	Unused            [4]uintptr
}

func CleanRam() error {
	cleanMutex.Lock()
	defer cleanMutex.Unlock()

	runtime.GC()
	debug.FreeOSMemory()

	switch runtime.GOOS {
	case "windows":
		return cleanRamWindows()
	case "linux":
		return cleanRamLinux()
	case "darwin":
		return cleanRamMacOS()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func cleanRamWindows() error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	ntdll := syscall.NewLazyDLL("ntdll.dll")

	currentProcess, _ := syscall.GetCurrentProcess()

	setProcessWorkingSetSizeProc := kernel32.NewProc("SetProcessWorkingSetSize")
	setProcessWorkingSetSizeProc.Call(uintptr(currentProcess), ^uintptr(0), ^uintptr(0))

	ntSetSystemInformationProc := ntdll.NewProc("NtSetSystemInformation")

	var cacheInfo SYSTEM_CACHE_INFORMATION
	cacheInfo.MinimumWorkingSet = ^uintptr(0)
	cacheInfo.MaximumWorkingSet = ^uintptr(0)

	ntSetSystemInformationProc.Call(
		uintptr(21),
		uintptr(unsafe.Pointer(&cacheInfo)),
		unsafe.Sizeof(cacheInfo),
	)

	command1 := uintptr(1)
	ntSetSystemInformationProc.Call(80, uintptr(unsafe.Pointer(&command1)), unsafe.Sizeof(command1))

	command2 := uintptr(2)
	ntSetSystemInformationProc.Call(80, uintptr(unsafe.Pointer(&command2)), unsafe.Sizeof(command2))

	command3 := uintptr(3)
	ntSetSystemInformationProc.Call(80, uintptr(unsafe.Pointer(&command3)), unsafe.Sizeof(command3))

	command4 := uintptr(4)
	ntSetSystemInformationProc.Call(80, uintptr(unsafe.Pointer(&command4)), unsafe.Sizeof(command4))

	exec.Command("powershell", "-Command", "Clear-RecycleBin -Force -ErrorAction SilentlyContinue").Run()

	exec.Command("rundll32.exe", "advapi32.dll,ProcessIdleTasks").Run()

	exec.Command("ipconfig", "/flushdns").Run()

	return nil
}

func cleanRamLinux() error {
	exec.Command("sync").Run()

	exec.Command("sh", "-c", "echo 1 > /proc/sys/vm/drop_caches").Run()
	exec.Command("sh", "-c", "echo 2 > /proc/sys/vm/drop_caches").Run()
	exec.Command("sh", "-c", "echo 3 > /proc/sys/vm/drop_caches").Run()

	exec.Command("sh", "-c", "echo 1 > /proc/sys/vm/compact_memory").Run()

	return nil
}

func cleanRamMacOS() error {
	exec.Command("purge").Run()
	exec.Command("sync").Run()
	exec.Command("dscacheutil", "-flushcache").Run()

	return nil
}
