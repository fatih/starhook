package git

import (
	"fmt"
	"os/exec"
)

type Client struct {
	dir string
}

func NewClient(repoDir string) (*Client, error) {
	g := &Client{
		dir: repoDir,
	}

	return g, nil
}

func (g *Client) Run(args ...string) ([]byte, error) {
	c := exec.Command("git", args...)
	if g.dir != "" {
		c.Dir = g.dir
	}

	out, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running git failed: %w (out: %q, args: %+v, dir: %s)",
			err, string(out), args, c.Dir)
	}

	return out, nil
}
