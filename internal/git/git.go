package git

import (
	"fmt"
	"os/exec"
)

type Client struct {
	Dir string
}

func (g *Client) Run(args ...string) ([]byte, error) {
	c := exec.Command("git", args...)
	if g.Dir != "" {
		c.Dir = g.Dir
	}

	out, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running git failed: %w (out:%s, args: %+v)",
			err, string(out), args)
	}

	return out, nil
}
