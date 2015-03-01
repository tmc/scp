// Package scp provides a simple interface to copying files over a
// go.crypto/ssh session.
package scp

import (
	"fmt"
	"io"
	"os"
	"path"

	"golang.org/x/crypto/ssh"
)

func Copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destinationPath string, session *ssh.Session) error {
	return copy(size, mode, fileName, contents, destinationPath, session)
}

func CopyPath(filePath, destinationPath string, session *ssh.Session) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return err
	}
	return copy(s.Size(), s.Mode().Perm(), path.Base(filePath), f, destinationPath, session)
}

func copy(size int64, mode os.FileMode, fileName string, contents io.Reader, destination string, session *ssh.Session) error {
	defer session.Close()
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%#o %d %s\n", mode, size, fileName)
		io.Copy(w, contents)
		fmt.Fprint(w, "\x00")
	}()
	cmd := fmt.Sprintf("scp -t %s", destination)
	if err := session.Run(cmd); err != nil {
		return err
	}
	return nil
}
