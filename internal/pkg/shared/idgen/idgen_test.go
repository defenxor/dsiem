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

package idgen

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"
)

func TestIdgen(t *testing.T) {
	var lock sync.Mutex
	m := map[string]bool{}
	g := new(errgroup.Group)
	for i := 0; i < 1000; i++ {
		g.Go(func() error {
			id, err := GenerateID()
			if err != nil {
				t.Error(err)
				return err
			}

			if strings.HasPrefix(id, "-") {
				return fmt.Errorf("id contain prefix '-', '%s'", id)
			}

			lock.Lock()
			_, ok := m[id]
			if ok {
				return fmt.Errorf("id '%s' already generated", id)
			}

			m[id] = true
			lock.Unlock()

			return nil
		})

	}

	if err := g.Wait(); err != nil {
		t.Error(err)
	}

	t.Logf("generated %d unique ids", len(m))

}
