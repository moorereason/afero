// Copyright © 2014 Steve Francia <spf@spf13.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memmapfs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
)

const FilePathSeparator = string(filepath.Separator)

type Fs struct {
	mu   sync.RWMutex
	data map[string]*FileData
	init sync.Once
}

func New() *Fs {
	return &Fs{}
}

func (m *Fs) getData() map[string]*FileData {
	m.init.Do(func() {
		m.data = make(map[string]*FileData)
		// Root should always exist, right?
		// TODO: what about windows?
		m.data[FilePathSeparator] = CreateDir(FilePathSeparator)
	})
	return m.data
}

func (*Fs) Name() string { return "MemMapFS" }

func (m *Fs) Create(name string) (afero.File, error) {
	name = normalizePath(name)
	m.mu.Lock()
	file := CreateFile(name)
	m.getData()[name] = file
	m.registerWithParent(file)
	m.mu.Unlock()
	return NewFileHandle(file), nil
}

func (m *Fs) unRegisterWithParent(fileName string) error {
	f, err := m.lockfreeOpen(fileName)
	if err != nil {
		return err
	}
	parent := m.findParent(f)
	if parent == nil {
		log.Panic("parent of ", f.Name(), " is nil")
	}

	parent.Lock()
	RemoveFromMemDir(parent, f)
	parent.Unlock()
	return nil
}

func (m *Fs) findParent(f *FileData) *FileData {
	pdir, _ := filepath.Split(f.Name())
	pdir = filepath.Clean(pdir)
	pfile, err := m.lockfreeOpen(pdir)
	if err != nil {
		return nil
	}
	return pfile
}

func (m *Fs) registerWithParent(f *FileData) {
	if f == nil {
		return
	}
	parent := m.findParent(f)
	if parent == nil {
		pdir := filepath.Dir(filepath.Clean(f.Name()))
		err := m.lockfreeMkdir(pdir, 0777)
		if err != nil {
			// log.Println("Mkdir error:", err)
			return
		}
		parent, err = m.lockfreeOpen(pdir)
		if err != nil {
			// log.Println("Open after Mkdir error:", err)
			return
		}
	}

	parent.Lock()
	InitializeDir(parent)
	AddToMemDir(parent, f)
	parent.Unlock()
}

func (m *Fs) lockfreeMkdir(name string, perm os.FileMode) error {
	name = normalizePath(name)
	x, ok := m.getData()[name]
	if ok {
		// Only return ErrFileExists if it's a file, not a directory.
		i := FileInfo{FileData: x}
		if !i.IsDir() {
			return os.ErrExist
		}
	} else {
		item := CreateDir(name)
		m.getData()[name] = item
		m.registerWithParent(item)
	}
	return nil
}

func (m *Fs) Mkdir(name string, perm os.FileMode) error {
	name = normalizePath(name)

	m.mu.RLock()
	_, ok := m.getData()[name]
	m.mu.RUnlock()
	if ok {
		return &os.PathError{Op: "mkdir", Path: name, Err: os.ErrExist}
	}

	m.mu.Lock()
	item := CreateDir(name)
	m.getData()[name] = item
	m.registerWithParent(item)
	m.mu.Unlock()

	m.Chmod(name, perm|os.ModeDir)

	return nil
}

func (m *Fs) MkdirAll(path string, perm os.FileMode) error {
	err := m.Mkdir(path, perm)
	if err != nil {
		if err.(*os.PathError).Err == os.ErrExist {
			return nil
		}
		return err
	}
	return nil
}

// Handle some relative paths
func normalizePath(path string) string {
	path = filepath.Clean(path)

	switch path {
	case ".":
		return FilePathSeparator
	case "..":
		return FilePathSeparator
	default:
		return path
	}
}

func (m *Fs) Open(name string) (afero.File, error) {
	f, err := m.open(name)
	if f != nil {
		return NewReadOnlyFileHandle(f), err
	}
	return nil, err
}

func (m *Fs) openWrite(name string) (afero.File, error) {
	f, err := m.open(name)
	if f != nil {
		return NewFileHandle(f), err
	}
	return nil, err
}

func (m *Fs) open(name string) (*FileData, error) {
	name = normalizePath(name)

	m.mu.RLock()
	f, ok := m.getData()[name]
	m.mu.RUnlock()
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	return f, nil
}

func (m *Fs) lockfreeOpen(name string) (*FileData, error) {
	name = normalizePath(name)
	f, ok := m.getData()[name]
	if ok {
		return f, nil
	} else {
		return nil, os.ErrNotExist
	}
}

func (m *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	chmod := false
	file, err := m.openWrite(name)
	if os.IsNotExist(err) && (flag&os.O_CREATE > 0) {
		file, err = m.Create(name)
		chmod = true
	}
	if err != nil {
		return nil, err
	}
	if flag == os.O_RDONLY {
		file = NewReadOnlyFileHandle(file.(*File).Data())
	}
	if flag&os.O_APPEND > 0 {
		_, err = file.Seek(0, os.SEEK_END)
		if err != nil {
			file.Close()
			return nil, err
		}
	}
	if flag&os.O_TRUNC > 0 && flag&(os.O_RDWR|os.O_WRONLY) > 0 {
		err = file.Truncate(0)
		if err != nil {
			file.Close()
			return nil, err
		}
	}
	if chmod {
		m.Chmod(name, perm)
	}
	return file, nil
}

func (m *Fs) Remove(name string) error {
	name = normalizePath(name)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.getData()[name]; ok {
		err := m.unRegisterWithParent(name)
		if err != nil {
			return &os.PathError{Op: "remove", Path: name, Err: err}
		}
		delete(m.getData(), name)
	} else {
		return &os.PathError{Op: "remove", Path: name, Err: os.ErrNotExist}
	}
	return nil
}

func (m *Fs) RemoveAll(path string) error {
	path = normalizePath(path)
	m.mu.Lock()
	m.unRegisterWithParent(path)
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	for p, _ := range m.getData() {
		if strings.HasPrefix(p, path) {
			m.mu.RUnlock()
			m.mu.Lock()
			delete(m.getData(), p)
			m.mu.Unlock()
			m.mu.RLock()
		}
	}
	return nil
}

func (m *Fs) Rename(oldname, newname string) error {
	oldname = normalizePath(oldname)
	newname = normalizePath(newname)

	if oldname == newname {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.getData()[oldname]; ok {
		m.mu.RUnlock()
		m.mu.Lock()
		m.unRegisterWithParent(oldname)
		fileData := m.getData()[oldname]
		delete(m.getData(), oldname)
		ChangeFileName(fileData, newname)
		m.getData()[newname] = fileData
		m.registerWithParent(fileData)
		m.mu.Unlock()
		m.mu.RLock()
	} else {
		return &os.PathError{Op: "rename", Path: oldname, Err: os.ErrNotExist}
	}
	return nil
}

func (m *Fs) Stat(name string) (os.FileInfo, error) {
	f, err := m.Open(name)
	if err != nil {
		return nil, err
	}
	fi := GetFileInfo(f.(*File).Data())
	return fi, nil
}

func (m *Fs) Chmod(name string, mode os.FileMode) error {
	name = normalizePath(name)

	m.mu.RLock()
	f, ok := m.getData()[name]
	m.mu.RUnlock()
	if !ok {
		return &os.PathError{Op: "chmod", Path: name, Err: os.ErrNotExist}
	}

	m.mu.Lock()
	SetMode(f, mode)
	m.mu.Unlock()

	return nil
}

func (m *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name = normalizePath(name)

	m.mu.RLock()
	f, ok := m.getData()[name]
	m.mu.RUnlock()
	if !ok {
		return &os.PathError{Op: "chtimes", Path: name, Err: os.ErrNotExist}
	}

	m.mu.Lock()
	SetModTime(f, mtime)
	m.mu.Unlock()

	return nil
}

func (m *Fs) List() {
	for _, x := range m.data {
		y := FileInfo{FileData: x}
		fmt.Println(x.Name(), y.Size())
	}
}

// func debugMemMapList(fs afero.Fs) {
// 	if x, ok := fs.(*Fs); ok {
// 		x.List()
// 	}
// }
