package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// commands lists the available commands and help topics printed in order.
var commands = []*Command{
	cmdDaemon,
	cmdWatch,
}

var defaultCommand = cmdWatch

var exitMu sync.Mutex
var exitStatus = 0

func setExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func main() {
	flag.Usage = usage
	if (len(os.Args) == 1) {
		os.Args = append(os.Args, defaultCommand.Name())
	}
	if strings.HasPrefix(os.Args[1], "-") {
		var a []string
		a = append(a, os.Args[0])
		a = append(a, defaultCommand.Name())
		a = append(a, os.Args[1:]...)
		os.Args = a
	}
	flag.Parse()
	log.SetFlags(0)

	args := flag.Args()
	if args[0] == "help" {
		help(args[1:])
		return
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			invokeCommand(cmd, args)
			exit()
			return
		}
	}

	if defaultCommand == nil {
	       fmt.Fprintf(os.Stderr, "jk: unknown subcommand %q\nRun 'jk help' for usage.\n", args[0])
	       setExitStatus(2)
	} else {
		var tmp []string
		tmp = append(tmp, defaultCommand.Name())
		tmp = append(tmp, args...)
		invokeCommand(defaultCommand, tmp)
	}
	exit()
}

func invokeCommand(cmd *Command, args []string) {
	cmd.Flag.Usage = func() { cmd.Usage() }
	if cmd.CustomFlags {
		args = args[1:]
	} else {
		cmd.Flag.Parse(args[1:])
		args = cmd.Flag.Args()
	}
	cmd.Run(cmd, args)
}

var atexitFuncs []func()

func atexit(f func()) {
	atexitFuncs = append(atexitFuncs, f)
}

func exit() {
	for _, f := range atexitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

func usage() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		help([]string{"testflag"})
		os.Exit(2)
	}
	printUsage(os.Stderr)
	os.Exit(2)
}

func printUsage(w io.Writer) {
	tmpl(w, usageTemplate, commands)
}

func help(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: jk help command\n\nToo many arguments given.\n")
		os.Exit(2)
	}

	arg := args[0]

	for _, cmd := range commands {
		if cmd.Name() == arg {
			tmpl(os.Stdout, helpTemplate, cmd)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic %#q. Run 'jk help'.\n", arg)
	os.Exit(2)
}

var helpTemplate = `{{if .Runnable}}usage: jk {{.UsageLine}}

{{end}}{{.Long | trim}}
`

var usageTemplate = `jk is a collection of tools for creating and observing failures in distributed systems

Usage:

	jk comand [arguments]

The commands are:
{{range .}}{{if .Runnable}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "jk help [command]" for more information about a command.

Additional help topics:
{{range .}}{{if not .Runnable}}
	{{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "jk help [topic]" for more information about that topic.

`

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}

	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}
