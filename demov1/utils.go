package demov1

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikelsr/bspl"
)

func getProtoFolder() (string, error) {
	_, fileName, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("Failed to locate project")
	}
	dir, err := filepath.Abs(filepath.Dir(fileName))
	if err != nil {
		return "", err
	}
	path := strings.Split(dir, string(os.PathSeparator))
	dir = "/" + filepath.Join(path[:len(path)-1]...)
	return filepath.Join(dir, "protocols"), nil
}

func getProtocol(filename string) bspl.Protocol {
	folder, err := getProtoFolder()
	if err != nil {
		panic(err)
	}
	reader, err := os.Open(filepath.Join(folder, filename))
	if err != nil {
		panic(err)
	}
	p, err := bspl.Parse(reader)
	if err != nil {
		panic(err)
	}
	return p
}
