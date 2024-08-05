package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Containers []containers
}

type containers struct {
	Memory string
}

var cfg *Config

func init() {
	checkErr(os.MkdirAll("./containers", 0700), "init containers dir")
	cfg = ReadConfig()
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
	checkErr(cmd.Start(), "cmd.start")

	checkErr(cmd.Wait(), "cmd.wait()")
}

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[1], os.Getpid())
	c := cfg.Containers[0]
	memoryMaxInt, err := strconv.Atoi(string(c.Memory[len(c.Memory)-3]))
	checkErr(err, "child(): atoi")
	switch c.Memory[len(c.Memory)-2] {
	case 'M':
		memoryMaxInt = memoryMaxInt * int(math.Pow(2, 20))
	case 'G':
		memoryMaxInt = memoryMaxInt * int(math.Pow(2, 30))
	}

	cg(memoryMaxInt)

	uid, err := uuid.NewRandom()
	checkErr(err, "child(): uuid")

	containerId := "f-" + uid.String()

	rootfsDir := filepath.Join("./containers", containerId)

	checkErr(syscall.Sethostname([]byte("container")), "child(): Sethostname")
	unzipImage(rootfsDir, "./ubuntu-base-20.04.1-base-amd64.tar.gz")

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	pivotRoot(rootfsDir)

	checkErr(syscall.Mount("proc", "proc", "proc", 0, ""), "child(): syscall mount")
	checkErr(cmd.Run(), "run child()")
	checkErr(syscall.Unmount("/proc", 0), "child(): syscall Unmount")
}

func ensureDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
}

func cg(memoryMax int) {
	cgroups := "/sys/fs/cgroup"
	fukumu := filepath.Join(cgroups, "Fukumu")

	ensureDir(fukumu)

	checkErr(os.WriteFile(filepath.Join(fukumu, "pids.max"), []byte("20"), 0700), "write pids.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700), "write cgroup.procs")
	checkErr(os.WriteFile(filepath.Join(fukumu, "memory.max"), []byte(strconv.Itoa(memoryMax)), 0700), "write memory.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "memory.min"), []byte(strconv.Itoa(6*int(math.Pow(2, 20)))), 0700), "write memory.max")
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

func ReadConfig() *Config {
	doc, err := os.ReadFile("fukumu.toml")
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config
	err = toml.Unmarshal([]byte(doc), &cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg
}

func checkErr(err error, context string) {
	if err != nil {
		log.Fatalf("%s failed: %v", context, err)
	}
}
