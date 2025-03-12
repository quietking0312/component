package mmen

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type Process struct {
	Pid     int
	Name    string
	Cmdline string
}

func getProcessesWindow() ([]Process, error) {
	var processes []Process
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for {
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
		processes = append(processes, Process{
			Pid:     int(entry.ProcessID),
			Name:    syscall.UTF16ToString(entry.ExeFile[:]),
			Cmdline: "",
		})
	}
	return processes, nil
}

func getProcessesLinux() ([]Process, error) {
	var processes []Process

	// 遍历 /proc 下的所有目录（PID）
	pids, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, pidDir := range pids {
		pidStr := pidDir.Name()

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// 读取进程名称（status 文件中的 Name 字段）
		statusPath := filepath.Join("/proc", pidStr, "status")
		statusData, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}

		name := getNameFromStatus(statusData)
		if name == "" {
			continue
		}

		// 读取命令行（cmdline 文件）
		cmdlinePath := filepath.Join("/proc", pidStr, "cmdline")
		cmdlineData, err := os.ReadFile(cmdlinePath)
		if err != nil {
			continue
		}

		processes = append(processes, Process{
			Pid:     pid,
			Name:    name,
			Cmdline: strings.TrimSpace(string(cmdlineData)),
		})
	}

	return processes, nil
}

func getNameFromStatus(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Name:") {
			return strings.TrimSpace(line[len("Name:"):])
		}
	}
	return ""
}

func getProcesses() ([]Process, error) {
	switch os.Getenv("GOOS") {
	case "windows":
		return getProcessesWindow()
	case "linux", "darwin":
		return getProcessesLinux()
	default:
		return nil, fmt.Errorf("unsupported os")
	}

}

//func readProcessMemory(pid int, addr uintptr, buf []byte) (int, error) {
//	hProcess, err := windows.OpenProcess(0x0010, false, uint32(pid))
//	if err != nil {
//		return 0, err
//	}
//	defer windows.CloseHandle(hProcess)
//	err = windows.ReadProcessMemory(hProcess, addr, buf, unsafe.Sizeof(buf), nil)
//	if err != nil {
//		return 0, err
//	}
//	return 0, nil
//}

func GetProcess() {
	process, err := getProcesses()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range process {
		fmt.Println(p.Pid, p.Name, p.Cmdline)
	}
}
