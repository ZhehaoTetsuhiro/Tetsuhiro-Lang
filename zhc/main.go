package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var (
		inFile  string
		outFile string
		asmFile string
		run     bool
	)
	i := 1
	for i < len(os.Args) {
		a := os.Args[i]
		switch a {
		case "-o":
			i++
			if i >= len(os.Args) {
				fatal("「-o」后无径")
			}
			outFile = os.Args[i]
		case "-s":
			i++
			if i >= len(os.Args) {
				fatal("「-s」后无径")
			}
			asmFile = os.Args[i]
		case "--run":
			run = true
		case "-h", "--help":
			usage()
			os.Exit(0)
		default:
			if strings.HasPrefix(a, "-") {
				fatal("未识之旗：「%s」", a)
			}
			if inFile != "" {
				fatal("唯纳一界：已得「%s」，又得「%s」", inFile, a)
			}
			inFile = a
		}
		i++
	}
	if inFile == "" {
		usage()
		os.Exit(2)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), filepath.Ext(inFile))
	}
	if asmFile == "" {
		asmFile = outFile + ".s"
	}

	src, err := compile(inFile)
	if err != nil {
		fatal("%s", err)
	}
	if err := os.WriteFile(asmFile, []byte(src), 0o644); err != nil {
		fatal("简牍难书：%s", err)
	}
	fmt.Printf("注曰「已成汇编：%s」\n", asmFile)

	// 注曰「以系统汇编器与链接器铸 AMD64 之 Linux 可执（静态）。」
	objFile := outFile + ".仿o"
	as := exec.Command("as", "--64", asmFile, "-o", objFile)
	as.Stdout = os.Stdout
	as.Stderr = os.Stderr
	if err := as.Run(); err != nil {
		fatal("汇编未成：%s", err)
	}
	defer os.Remove(objFile)
	ld := exec.Command("ld", "-static", "-o", outFile, objFile)
	ld.Stdout = os.Stdout
	ld.Stderr = os.Stderr
	if err := ld.Run(); err != nil {
		fatal("链接未成：%s", err)
	}
	if err := os.Chmod(outFile, 0o755); err != nil {
		fatal("不可授行之权：%s", err)
	}
	fmt.Printf("注曰「已铸可执：%s（AMD64 · Linux）」\n", outFile)

	if run {
		fmt.Println("注曰「运行——」")
		// 注曰「器名无径者，exec 循 PATH 寻之，而当居之地不在 PATH，故冠「./」以指其居」
		runExe := outFile
		if !strings.ContainsRune(runExe, filepath.Separator) {
			runExe = "./" + runExe
		}
		cmd := exec.Command(runExe)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				os.Exit(ee.ExitCode())
			}
			fatal("运行未毕：%s", err)
		}
	}
}

func usage() {
	fmt.Println(`哲浩之谕（zhc · AMD64）
用法：zhc <界.浩> [-o 出] [-s 出.s] [--run]

  将界经编译为 AMD64（x86-64）汇编，经系统汇编器
  与链接器铸为静态链接之 Linux 可执行文件。
  界以「哲浩御世」开篇；承《名》则先寻《名》.哲（典册），
  次寻《名》.浩（经卷）。
  --run  铸后即行。`)
}

// 注曰「compile loads a program and emits assembly, converting any panic (from」
// 注曰「defensive validation) into a proper error.」
func compile(path string) (src string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	prog, err := loadProgram(path)
	if err != nil {
		return "", err
	}
	return compileProgram(prog), nil
}

func fatal(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "劫：%s\n", fmt.Sprintf(format, a...))
	os.Exit(1)
}
