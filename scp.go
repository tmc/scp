// Package scp provides a simple interface to copying files over a
// go.crypto/ssh session.
package scp

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

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
func CopyLocalFileToRemotePath(localFile, remotePath string, client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	f, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return err
	}
	return copy(s.Size(), s.Mode().Perm(), path.Base(localFile), f, remotePath, session)
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

func GetRemoteFileAccessRights(remoteFile string, client *ssh.Client) (os.FileMode, error) {
	session, err := client.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()
	bs, err := session.Output("/usr/bin/stat --format=%a " + remoteFile)
	if err != nil {
		return 0, err
	}
	s := string(bs)
	s = strings.TrimSpace(s)
	mode64, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, err
	}
	var mode os.FileMode = os.FileMode(mode64)
	return mode, nil

}
func CopyRemoteFileToLocalPath(remoteFile, localPath string, client *ssh.Client) error {
	mode, err := GetRemoteFileAccessRights(remoteFile, client)
	if err != nil {
		return err
	}

	fileName := filepath.Base(remoteFile)
	localFile, err := os.OpenFile(localPath+string(filepath.Separator)+fileName, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return err
	}
	defer localFile.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = localFile

	err = session.Run("/bin/cat " + remoteFile)
	if err != nil {
		return err
	}
	return nil

}
