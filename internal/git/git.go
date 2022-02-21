package git

import (
	"fmt"
	"os"
	"os/exec"
)

type Client struct {
	Dir string
}

func NewClient(repoDir string) (*Client, error) {
	g := &Client{
		Dir: repoDir,
	}

	_, err := os.Stat(repoDir)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *Client) Run(args ...string) ([]byte, error) {
	c := exec.Command("git", args...)
	if g.Dir != "" {
		c.Dir = g.Dir
	}

	out, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running git failed: %w (out: %q, args: %+v, dir: %s)",
			err, string(out), args, c.Dir)
	}

	return out, nil
}
