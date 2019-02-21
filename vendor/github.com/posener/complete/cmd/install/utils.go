package install

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func lineInFile(name string, lookFor string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	r := bufio.NewReader(f)
	prefix := []byte{}
	for {
		line, isPrefix, err := r.ReadLine()
		if err == io.EOF {
			return false
		}
		if err != nil {
			return false
		}
		if isPrefix {
			prefix = append(prefix, line...)
			continue
		}
		line = append(prefix, line...)
		if string(line) == lookFor {
			return true
		}
		prefix = prefix[:0]
	}
}

func createFile(name string, content string) error {
	// make sure file directory exists
	if err := os.MkdirAll(filepath.Dir(name), 0775); err != nil {
		return err
	}

	// create the file
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()

	// write file content
	_, err = f.WriteString(fmt.Sprintf("%s\n", content))
	return err
}

func appendToFile(name string, content string) error {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("\n%s\n", content))
	return err
}

func removeFromFile(name string, content string) error {
	backup := name + ".bck"
	err := copyFile(name, backup)
	if err != nil {
		return err
	}
	temp, err := removeContentToTempFile(name, content)
	if err != nil {
		return err
	}

	err = copyFile(temp, name)
	if err != nil {
		return err
	}

	return os.Remove(backup)
}

func removeContentToTempFile(name, content string) (string, error) {
	rf, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer rf.Close()
	wf, err := ioutil.TempFile("/tmp", "complete-")
	if err != nil {
		return "", err
	}
	defer wf.Close()

	r := bufio.NewReader(rf)
	prefix := []byte{}
	for {
		line, isPrefix, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if isPrefix {
			prefix = append(prefix, line...)
			continue
		}
		line = append(prefix, line...)
		str := string(line)
		if str == content {
			continue
		}
		_, err = wf.WriteString(str + "\n")
		if err != nil {
			return "", err
		}
		prefix = prefix[:0]
	}
	return wf.Name(), nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
