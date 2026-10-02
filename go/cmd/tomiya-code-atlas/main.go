package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeHelp(stdout)
		return 0
	}

	switch args[0] {
	case "--help", "-h", "help":
		writeHelp(stdout)
		return 0
	case "--version", "-V", "version":
		fmt.Fprintf(stdout, "Tomiya Code Atlas (Go) %s\n", version)
		return 0
	default:
		fmt.Fprintf(stderr, "[エラー] まだ利用できないコマンドです: %s\n", args[0])
		fmt.Fprintln(stderr, "利用可能なコマンドは --help で確認してください。")
		return 2
	}
}

func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "Tomiya Code Atlas (Go)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "使用方法: tomiya-code-atlas [コマンド]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "コマンド:")
	fmt.Fprintln(w, "  --help, -h     この案内を表示")
	fmt.Fprintln(w, "  --version, -V  バージョンを表示")
}
