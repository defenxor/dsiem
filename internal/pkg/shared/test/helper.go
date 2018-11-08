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

package test

import (
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

//DirEnv get the root app directory and setup log for testing
func DirEnv() (dir string, err error) {
	dir, err = fs.GetDir(true)
	if err == nil {
		err = log.Setup(false)
	}
	return
}
