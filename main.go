package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/uuid"
	"github.com/pbnjay/memory"
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
	case "clean":
		os.RemoveAll(CreateFukumuPath())
	default:
		log.Fatal("bad command")
	}
}

func run() {
	fmt.Printf("Running %v as %d\n", os.Args[2], os.Getpid())

	mMax, mMin := SetMemory()
	limit := syscall.Rlimit{
		Cur: uint64(mMin),
		Max: uint64(mMax),
	}

	checkErr(syscall.Setrlimit(syscall.RLIMIT_AS, &limit), "syscall.setrlimit()")

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	checkErr(cmd.Start(), "cmd.start")

	go func() {
		<-sigChan
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill process: %v\n", err)
		}
	}()

	err := cmd.Wait()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			fmt.Printf("Process exited with status: %d\n", ws.ExitStatus())
			if ws.Signaled() {
				fmt.Printf("Process was killed by signal: %s\n", ws.Signal())
			}
		}
		os.Exit(1)
	}
}

func child() {
	fmt.Printf("Running %v as %d\n", os.Args[1], os.Getpid())
	mMax, mMin := SetMemory()
	cg(mMax, mMin)

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

func cg(memoryMax, memoryMin int) {
	fukumu := CreateFukumuPath()
	ensureDir(fukumu)
	if memoryMax >= int(memory.FreeMemory()) {
		log.Fatal("Memory allocated bigger than memory free in system")
	}
	if memoryMax < memoryMin {
		fmt.Println("memoryMax:", memoryMax, "memoryMin:", memoryMin)
		log.Fatal("memoryMax minor than memoryMin")
	}

	checkErr(os.WriteFile(filepath.Join(fukumu, "pids.max"), []byte("20"), 0700), "write pids.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700), "write cgroup.procs")
	checkErr(os.WriteFile(filepath.Join(fukumu, "memory.max"), []byte(strconv.Itoa(memoryMax)), 0700), "write memory.max")
	checkErr(os.WriteFile(filepath.Join(fukumu, "memory.min"), []byte(strconv.Itoa(memoryMin)), 0700), "write memory.min")
	checkErr(os.WriteFile(filepath.Join(fukumu, "memory.high"), []byte(strconv.Itoa(memoryMax)), 0700), "write memory.high")
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

func SetMemory() (int, int) {
	// TODO: iterate through the containers
	c := cfg.Containers[0]
	fmt.Println("contianer memory", c.Memory)
	memoryMax, err := strconv.Atoi(string(c.Memory[:len(c.Memory)-2]))
	fmt.Println("memoryMaxInt", memoryMax)
	checkErr(err, "child(): atoi")
	switch c.Memory[len(c.Memory)-2] {
	case 'M':
		memoryMax = memoryMax * 1024 * 1024
	case 'G':
		memoryMax = memoryMax * 1024 * 1024 * 1024
	}

	memoryMin := 2 * 1024 * 1024 * 1024 // 2GB

	return memoryMax, memoryMin
}

func CreateFukumuPath() string {
	cgroups := "/sys/fs/cgroup"
	filepath.Join(cgroups)
	return filepath.Join(cgroups, "fukumu")
}

func checkErr(err error, context string) {
	if err != nil {
		log.Fatalf("%s failed: %v", context, err)
	}
}
