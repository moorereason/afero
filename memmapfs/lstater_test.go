// Copyright ©2018 Steve Francia <spf@spf13.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memmapfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/afero/basepathfs"
	"github.com/spf13/afero/cowfs"
	"github.com/spf13/afero/fsutil"
	"github.com/spf13/afero/osfs"
	"github.com/spf13/afero/rofs"
)

func TestLstatIfPossible(t *testing.T) {
	wd, _ := os.Getwd()
	defer func() {
		os.Chdir(wd)
	}()

	osFs := &osfs.OsFs{}

	workDir, err := fsutil.TempDir(osFs, "", "afero-lstate")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		osFs.RemoveAll(workDir)
	}()

	memWorkDir := "/lstate"

	memFs := NewMemMapFs()
	overlayFs1 := cowfs.NewCopyOnWriteFs(osFs, memFs)
	overlayFs2 := cowfs.NewCopyOnWriteFs(memFs, osFs)
	overlayFsMemOnly := cowfs.NewCopyOnWriteFs(memFs, NewMemMapFs())
	basePathFs := basepathfs.NewBasePathFs(osFs, workDir)
	basePathFsMem := basepathfs.NewBasePathFs(memFs, memWorkDir)
	roFs := rofs.NewReadOnlyFs(osFs)
	roFsMem := rofs.NewReadOnlyFs(memFs)

	pathFileMem := filepath.Join(memWorkDir, "aferom.txt")

	fsutil.WriteFile(osFs, filepath.Join(workDir, "afero.txt"), []byte("Hi, Afero!"), 0777)
	fsutil.WriteFile(memFs, filepath.Join(pathFileMem), []byte("Hi, Afero!"), 0777)

	os.Chdir(workDir)
	if err := os.Symlink("afero.txt", "symafero.txt"); err != nil {
		t.Fatal(err)
	}

	pathFile := filepath.Join(workDir, "afero.txt")
	pathSymlink := filepath.Join(workDir, "symafero.txt")

	checkLstat := func(l afero.Lstater, name string, shouldLstat bool) os.FileInfo {
		statFile, isLstat, err := l.LstatIfPossible(name)
		if err != nil {
			t.Fatalf("Lstat check failed: %s", err)
		}
		if isLstat != shouldLstat {
			t.Fatalf("Lstat status was %t for %s", isLstat, name)
		}
		return statFile
	}

	testLstat := func(l afero.Lstater, pathFile, pathSymlink string) {
		shouldLstat := pathSymlink != ""
		statRegular := checkLstat(l, pathFile, shouldLstat)
		statSymlink := checkLstat(l, pathSymlink, shouldLstat)
		if statRegular == nil || statSymlink == nil {
			t.Fatal("got nil FileInfo")
		}

		symSym := statSymlink.Mode()&os.ModeSymlink == os.ModeSymlink
		if symSym == (pathSymlink == "") {
			t.Fatal("expected the FileInfo to describe the symlink")
		}

		_, _, err := l.LstatIfPossible("this-should-not-exist.txt")
		if err == nil || !os.IsNotExist(err) {
			t.Fatalf("expected file to not exist, got %s", err)
		}
	}

	testLstat(osFs, pathFile, pathSymlink)
	testLstat(overlayFs1, pathFile, pathSymlink)
	testLstat(overlayFs2, pathFile, pathSymlink)
	testLstat(basePathFs, "afero.txt", "symafero.txt")
	testLstat(overlayFsMemOnly, pathFileMem, "")
	testLstat(basePathFsMem, "aferom.txt", "")
	testLstat(roFs, pathFile, pathSymlink)
	testLstat(roFsMem, pathFileMem, "")
}
