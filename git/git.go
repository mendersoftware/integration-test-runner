package git

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/mendersoftware/integration-test-runner/logger"
)

var dryRunMode bool

func init() {
	dryRunMode = false
}

// SetDryRunMode sets the dry run mode
func SetDryRunMode(value bool) {
	dryRunMode = value
}

// Cmd is a git command
type Cmd struct {
	Dir     string
	Args    []string
	Process *cmdProcess
	cmd     *exec.Cmd
	out     []byte
	err     error
}

type cmdProcess struct{}

func (p *cmdProcess) Kill() error {
	return nil
}

// With sets the git command state
func (g *Cmd) With(s *State) *Cmd {
	g.Dir = s.Dir
	return g
}

// State holds the git command state
type State struct {
	Dir string
}

// Cleanup cleans up the statee
func (s *State) Cleanup() {
	if s.Dir != "" {
		os.RemoveAll(s.Dir)
	}
}

// Commands runs multiple git commands
func Commands(cmds ...*Cmd) (*State, error) {
	tdir, err := os.MkdirTemp("", "gitcmd")
	if err != nil {
		return &State{}, err
	}
	s := &State{Dir: tdir}
	err = CommandsWithState(s, cmds...)
	return s, err
}

func CommandsWithState(s *State, cmds ...*Cmd) error {
	for _, cmd := range cmds {
		cmd.Dir = s.Dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrapf(err,
				"git command: %s returned error:\n%s",
				cmd.cmd.Args, out)
		}
	}
	return nil
}

// Command creates a new git command
func Command(args ...string) *Cmd {
	return &Cmd{
		cmd:     exec.Command("git", args...),
		Args:    args,
		Process: &cmdProcess{},
	}
}

// Run runs a git command
func (g *Cmd) Run() error {
	if dryRunMode {
		msg := fmt.Sprintf("git.Run: %s", g.cmd)
		logger.GetRequestLogger().Push(msg)
		g.out = []byte{}
		return nil
	}
	if g.Dir != "" {
		g.cmd.Dir = g.Dir
	}
	out, err := g.cmd.CombinedOutput()
	g.out = out
	g.err = err
	if err != nil {
		return fmt.Errorf("%v returned error: %s: %s", g.cmd.Args, out, err.Error())
	}
	return nil
}

func (g *Cmd) CombinedOutput() ([]byte, error) {
	_ = g.Run()
	return g.out, g.err
}
