package pitreos

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type FileOps struct {
	filePath     string
	readWrite    bool
	file         *os.File
	lock         sync.Mutex
	isAppendOnly bool
	originalSize int64

	extentsLoaded bool
}

func NewFileOps(filePath string, readWrite bool) *FileOps {
	return &FileOps{
		readWrite: readWrite,
		filePath:  filePath,
	}
}

func (f *FileOps) Open() error {
	perms := os.O_RDONLY
	if f.readWrite {
		perms = os.O_RDWR | os.O_CREATE
	}
	fl, err := os.OpenFile(f.filePath, perms, 0644)
	if err != nil {
		return err
	}

	f.file = fl

	return nil
}

func (f *FileOps) Close() error {
	if f.file == nil {
		return fmt.Errorf("file not currently open, %q", f.filePath)
	}

	return f.file.Close()
}

func (f *FileOps) Truncate(size int64) error { return f.file.Truncate(size) }

func (f *FileOps) wipeChunk(offset int64, length int64) error {
	// TODO: fill with zeroes if the FS underneath doesn't support FIBMAP
	var zerobytes = make([]byte, length)
	return f.writeChunkToFile(offset, zerobytes)
}

func (f *FileOps) writeChunkToFile(offset int64, s []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	_, err := f.file.Seek(offset, 0)
	if err != nil {
		return err
	}

	_, err = f.file.Write(s)
	return err
}

func (f *FileOps) getLocalChunk(offset int64, size int64) (data []byte, empty bool, err error) {
	data = make([]byte, size)

	f.lock.Lock()
	defer f.lock.Unlock()

	_, err = f.file.Seek(offset, 0)
	if err != nil {
		return data, empty, fmt.Errorf("seek error: %s", err)
	}

	_, err = f.file.Read(data)
	if err != nil {
		return data, empty, fmt.Errorf("read error: %s", err)
	}

	if isEmptyChunk(data) {
		empty = true
	}

	return data, empty, nil
}

func isEmptyChunk(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}

func getDirFiles(directory string) (fileNames []string, err error) {
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return err
		}

		// FIXME: Add support for symbolic links, other special files like FIFO and weird oddities.

		fileNames = append(fileNames, path)
		return nil
	})
	return
}

func stringarrayContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
