// Package processbuilder provides the java.lang.ProcessBuilder surface used
// by portable Clojure libraries to launch child processes.
package processbuilder

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"

	"github.com/glojurelang/glojure/pkg/lang"
	"github.com/glojurelang/glojure/pkg/pkgmap"
)

type Redirect string

const Inherit Redirect = "INHERIT"

type Environment struct{ values map[string]string }

func (e *Environment) Put(key, value any) any {
	e.values[fmt.Sprint(key)] = fmt.Sprint(value)
	return nil
}

type ProcessBuilder struct {
	command       []string
	environment   *Environment
	inheritStderr bool
}

type Process struct {
	exit int32
	out  []byte
	err  []byte
}

func New(command any) *ProcessBuilder {
	values := []string{}
	for items := lang.Seq(command); items != nil; items = items.Next() {
		values = append(values, fmt.Sprint(items.First()))
	}
	return &ProcessBuilder{
		command:     values,
		environment: &Environment{values: map[string]string{}},
	}
}

func (b *ProcessBuilder) Environment() *Environment { return b.environment }

func (b *ProcessBuilder) RedirectError(value any) *ProcessBuilder {
	b.inheritStderr = value == Inherit
	return b
}

func (b *ProcessBuilder) Start() *Process {
	if len(b.command) == 0 {
		panic("ProcessBuilder: empty command")
	}
	command := exec.Command(b.command[0], b.command[1:]...)
	command.Env = os.Environ()
	for key, value := range b.environment.values {
		command.Env = append(command.Env, key+"="+value)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	if b.inheritStderr {
		command.Stderr = os.Stderr
	} else {
		command.Stderr = &stderr
	}
	exit := int32(0)
	if err := command.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exit = int32(exitError.ExitCode())
		} else {
			panic(err)
		}
	}
	return &Process{
		exit: exit,
		out:  stdout.Bytes(),
		err:  stderr.Bytes(),
	}
}

func (p *Process) WaitFor() int32                { return p.exit }
func (p *Process) GetInputStream() io.ReadCloser { return io.NopCloser(bytes.NewReader(p.out)) }
func (p *Process) GetErrorStream() io.ReadCloser { return io.NopCloser(bytes.NewReader(p.err)) }

// Link gives embedders an explicit package-retention reference.
func Link() {}

func init() {
	pkgmap.SetHostClassPackage("ProcessBuilder", "java.lang")
	pkgmap.SetHostClass("ProcessBuilder",
		lang.NewClass(reflect.TypeOf((*ProcessBuilder)(nil)), "java.lang.ProcessBuilder"))
	lang.RegisterHostConstructor("java.lang.ProcessBuilder", lang.FnFunc1(func(command any) any {
		return New(command)
	}))

	pkgmap.SetHostClassPackage("ProcessBuilder$Redirect", "java.lang")
	pkgmap.SetHostClass("ProcessBuilder$Redirect",
		lang.NewClass(reflect.TypeOf(Redirect("")), "java.lang.ProcessBuilder$Redirect"))
	for _, prefix := range []string{"ProcessBuilder$Redirect", "java.lang.ProcessBuilder$Redirect"} {
		pkgmap.Set(prefix+".INHERIT", Inherit)
	}
}
