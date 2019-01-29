eexec is a wrapper for Go's standard exec package with convenient process respawn functionality.

Usage example.
Next example respawns command execution 3 times if it exits with non-zero status code.
```
package main

import (
	"io"
	"os"

	"github.com/vchimishuk/eexec"
)

func main() {
	c, err := eexec.NewCommand("date", []string{"--invalid"}, nil)
	if err != nil {
		panic(err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s)
	c.RelaySignals(s)

	c.Start(eexec.RespawnPolicy{Times: 3})

	go io.CopyBuffer(os.Stdout, c.StdOut(), make([]byte, 1024))
	go io.CopyBuffer(os.Stderr, c.StdErr(), make([]byte, 1024))

	err = c.Wait()
	if err != nil {
		panic(err)
	}
}
```
