package util

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"io/ioutil"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

// Command is a struct containing the details of an external command to be executed
type Command struct {
	attempts           int
	Errors             []error
	Dir                string
	Name               string
	Args               []string
	ExponentialBackOff *backoff.ExponentialBackOff
	Timeout            time.Duration
	Verbose            bool
	Quiet              bool
}

// CommandInterface defines the interface for a Command
//go:generate pegomock generate github.com/jenkins-x/jx/pkg/util CommandInterface
type CommandInterface interface {
	DidError() bool
	DidFail() bool
	Error() error
	IsVerbose() bool
	IsQuiet() bool
	Run() (string, error)
	RunWithoutRetry() (string, error)
}

// Attempts The number of times the command has been executed
func (c *Command) Attempts() int {
	return c.attempts
}

// DidError returns a boolean if any error occurred in any execution of the command
func (c *Command) DidError() bool {
	if len(c.Errors) > 0 {
		return true
	}
	return false
}

// DidFail returns a boolean if the command could not complete (errored on every attempt)
func (c *Command) DidFail() bool {
	if len(c.Errors) == c.attempts {
		return true
	}
	return false
}

// Error returns the last error
func (c *Command) Error() error {
	if len(c.Errors) > 0 {
		return c.Errors[len(c.Errors)-1]
	}
	return nil
}

// IsVerbose Evaluates options and returns if Command is set to verbose
func (c *Command) IsVerbose() bool {
	return c.Verbose
}

// IsQuiet Evaluates options and returns if Command is set to quiet
func (c *Command) IsQuiet() bool {
	if c.IsVerbose() {
		return false
	}
	if c.Quiet {
		return true
	}
	return false
}

// Run Execute the command and block waiting for return values
func (c *Command) Run() (string, error) {
	os.Setenv("PATH", PathWithBinary(c.Dir))
	var r string
	var e error

	f := func() error {
		r, e = c.run()
		c.attempts++
		if e != nil {
			c.Errors = append(c.Errors, e)
			return e
		}
		return nil
	}

	c.ExponentialBackOff = backoff.NewExponentialBackOff()
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Minute
	}
	c.ExponentialBackOff.MaxElapsedTime = c.Timeout
	c.ExponentialBackOff.Reset()
	err := backoff.Retry(f, c.ExponentialBackOff)
	if err != nil {
		return "", err
	}
	return r, nil
}

// RunWithoutRetry Execute the command without retrying on failure and block waiting for return values
func (c *Command) RunWithoutRetry() (string, error) {
	os.Setenv("PATH", PathWithBinary(c.Dir))
	var r string
	var e error

	r, e = c.run()
	c.attempts++
	if e != nil {
		c.Errors = append(c.Errors, e)
	}
	return r, e
}

func (c *Command) run() (string, error) {
	e := exec.Command(c.Name, c.Args...)
	if c.Dir != "" {
		e.Dir = c.Dir
	}
	if c.IsQuiet() {
		e.Stdout = ioutil.Discard
		e.Stderr = ioutil.Discard
	}
	data, err := e.CombinedOutput()
	output := string(data)
	text := strings.TrimSpace(output)
	if err != nil {
		return text, errors.Wrapf(err, "failed to run '%s %s' command in directory '%s', output: '%s'",
			c.Name, strings.Join(c.Args, " "), c.Dir, text)
	}
	if c.IsVerbose() {
		log.Infoln(output)
	}
	return text, err
}

// PathWithBinary Sets the $PATH variable. Accepts an optional slice of strings containing paths to add to $PATH
func PathWithBinary(paths ...string) string {
	path := os.Getenv("PATH")
	binDir, _ := BinaryLocation()
	answer := path + string(os.PathListSeparator) + binDir
	mvnBinDir, _ := MavenBinaryLocation()
	if mvnBinDir != "" {
		answer += string(os.PathListSeparator) + mvnBinDir
	}
	for _, p := range paths {
		answer += string(os.PathListSeparator) + p
	}
	return answer
}
