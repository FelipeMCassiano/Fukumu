package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/uuid"
)

func init() {
	checkErr(os.MkdirAll("./containers", 0700), "init containers dir")
}

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

// go:embed ubuntu-base-20.04.1-base-amd64.tar.gz
var f embed.FS

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[2], os.Getpid())

	cg()

	uid, err := uuid.NewRandom()
	checkErr(err, "child(): uuid")

	containerId := "f-" + uid.String()

	rootfsDir := filepath.Join("./containers", containerId)

	checkErr(syscall.Sethostname([]byte("container")), "child(): syscall Sethostname")
	unzipImage(rootfsDir, "./ubuntu-base-20.04.1-base-amd64.tar.gz")
	checkErr(syscall.Mount("proc", "proc", "proc", 0, ""), "child(): syscall mount")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	pivotRoot(rootfsDir)

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

	ensureDir(fukumu)

	checkErr(os.WriteFile(filepath.Join(fukumu, "pids.max"), []byte("20"), 0700), "write pids.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700), "write cgroup.procs")
}

func pivotRoot(newRoot string) {
	checkErr(syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""), "pivotRoot(): syscall mount")

	putOld := filepath.Join(newRoot, ".put_old")
	checkErr(os.MkdirAll(putOld, 0700), "pivotRoot(): mkdirall")

	checkErr(syscall.PivotRoot(newRoot, putOld), "pivotRoot(): pivotRoot")
	checkErr(syscall.Chdir("/"), "pivotRoot(): chrdir")
}

func unzipImage(dest, src string) {
	checkErr(os.MkdirAll(dest, 0700), "unzipImage: mkdirall")

	cmd := exec.Command("tar", []string{"-xzf", src, "-C", dest}...)
	checkErr(cmd.Run(), "unzipImage: cmd run")
}

func checkErr(err error, context string) {
	if err != nil {
		log.Fatalf("%s failed: %v", context, err)
	}
}
