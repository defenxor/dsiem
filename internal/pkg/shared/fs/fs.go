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
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/kardianos/osext"
)

// FileExist check if path exist
func FileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetDir returns the program root directory
func GetDir(devEnv bool) (string, error) {
	dir, err := osext.ExecutableFolder()
	if devEnv {
		keyword := "dsiem"
		wd, _ := os.Getwd()
		if i := strings.Index(wd, keyword); i > -1 {
			dir = wd[:i+len(keyword)]
		}
	}
	return dir, err
}

// AppendToFile write s to the end of filename
func AppendToFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s + "\n")
	return err
}

// OverwriteFile truncate filename and write s into it
func OverwriteFile(s string, filename string) error {
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return Write(s, f)
}

func Write(s string, w io.StringWriter) error {
	_, err := w.WriteString(s + "\n")
	return err
}

// OverwriteFileBytes truncate filename and write b []bytes into it
func OverwriteFileBytes(b []byte, filename string) error {
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteBytes(b, f)
}

func WriteBytes(b []byte, w io.Writer) error {
	_, err := w.Write(b)
	return err
}

// OverwriteFileValueIndent marshall v into indented json, then truncate the file filename and write the marshalled bytes into it
func OverwriteFileValueIndent(v interface{}, filename string) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer f.Close()
	return WriteBytes(b, f)
}

// EnsureDir creates directory if it doesnt exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, os.FileMode(0700))
}
