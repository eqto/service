package service

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"log"
)

var (
	args    map[string]string
	stopFn  func() error
	panicFn func(args ...interface{})
)

func Get(name string) string {
	parseArgs()
	if arg, ok := args[name]; ok {
		return arg
	}
	return ``
}

func GetInt(name string) int {
	parseArgs()
	if arg, ok := args[name]; ok {
		i, e := strconv.Atoi(arg)
		if e != nil {
			return 0
		}
		return i
	}
	return 0
}

func HandlePanic() {
	if r := recover(); r != nil {
		if panicFn != nil {
			panicFn(recover())
		} else {
			log.Println(recover())
		}
	}
}

func OnPanic(fn func(args ...interface{})) {
	panicFn = fn
}

func OnStop(fn func() error) {
	stopFn = fn
}

func Filename() string {
	_, file := filepath.Split(os.Args[0])
	return file
}

func parseArgs() {
	if args != nil {
		return
	}
	args = make(map[string]string)
	for i, arg := range os.Args {
		if i > 1 {
			if strings.HasPrefix(arg, `--`) {
				split := strings.SplitN(arg, `=`, 2)
				if len(split) == 2 {
					args[strings.TrimSpace(split[0][2:])] = strings.TrimSpace(split[1])
				}
			}
		}
	}
}

func Run(fn func() error) error {
	filename := Filename()

	cmd, ok := arg(0)
	if !ok {
		println(fmt.Sprintf(`Usage: %s [start/stop/run]`, filename))
		return nil
	}
	switch cmd {
	case `start`:
		return start(filename, append([]string{`run`}, os.Args[1:]...)...)
	case `stop`:
		return stop(filename)
	case `run`:
		parseArgs()
		go fn()
		exitSignal := make(chan os.Signal, 1)
		signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGQUIT)
		<-exitSignal
		if stopFn != nil {
			if e := stopFn(); e != nil {
				return e
			}
		}
		return nil
	default:
		println(fmt.Sprintf(`Usage: %s [start/stop/run]`, filename))
	}
	return nil
}

func arg(idx int) (string, bool) {
	if idx < len(os.Args)-1 {
		return os.Args[idx+1], true
	}
	return ``, false
}

func start(filename string, args ...string) error {
	cmd := exec.Command(`./`+filename, args...)
	if e := cmd.Start(); e != nil {
		return e
	}
	pidfile := filename + `.pid`
	if e := ioutil.WriteFile(pidfile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); e != nil {
		return e
	}
	return nil
}

func stop(filename string) error {
	pidfile := filename + `.pid`
	data, e := ioutil.ReadFile(pidfile)
	if e != nil {
		return e
	}
	pid, e := strconv.Atoi(strings.TrimSpace(string(data)))
	if e != nil {
		return e
	}
	proc, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return proc.Signal(os.Interrupt)
}
