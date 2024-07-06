package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func main() {
	command := os.Args[1]
	switch command {
	case "run":
		run()
	case "child":
		child()
	default:
		log.Fatal("bad command")
	}
}

func run() {
	fmt.Printf("Running %v as %d\n", os.Args[2], os.Getpid())

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	checkErr(cmd.Run(), "run run()")
}

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[2], os.Getpid())

	cg()

	syscall.Sethostname([]byte("container"))
	syscall.Chroot("/")
	syscall.Chdir("/")
	syscall.Mount("proc", "proc", "proc", 0, "")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	checkErr(cmd.Run(), "run child()")
	syscall.Unmount("/proc", 0)
}

func ensureDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
}

func cg() {
	cgroups := "/sys/fs/cgroup"
	fukumu := filepath.Join(cgroups, "Fukumu")

	// Ensure the cgroup directory exists
	ensureDir(fukumu)

	// Write configuration files for cgroup v2
	checkErr(os.WriteFile(filepath.Join(fukumu, "pids.max"), []byte("20"), 0700), "write pids.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700), "write cgroup.procs")
}

func checkErr(err error, context string) {
	if err != nil {
		log.Fatalf("%s failed: %v", context, err)
	}
}
