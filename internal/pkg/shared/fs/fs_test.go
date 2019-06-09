// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package fs

import (
	"os"
	"path"
	"testing"
)

func TestFS(t *testing.T) {
	_, err := GetDir(true)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := GetDir(true)
	if err != nil || dir == "" {
		t.Fatal("expected to obtain program root directory")
	}

	tmpDir := path.Join(os.TempDir(), "dsiem")
	if err := EnsureDir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := path.Join(tmpDir, "file.txt")
	if err := AppendToFile("test", tmpFile); err != nil {
		t.Fatal(err)
	}

	if !FileExist(tmpFile) {
		t.Fatal("test file" + tmpFile + " doesnt exist")
	}

	if err := AppendToFile("test", "/proc"); err == nil {
		t.Fatal("o rly?")
	}
	if err := OverwriteFile("test", tmpFile); err != nil {
		t.Fatal(err)
	}
	if err := OverwriteFile("test", "/proc"); err == nil {
		t.Fatal("o rly?")
	}
}
