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

package dpluger

import "reflect"

type pluginTemplate struct {
	Plugin        Plugin
	Creator       string
	CreateDate    string
	Ref           tsvRef
	SIDField      string
	SIDFieldPlain string
	SIDListGroup  []PluginSIDWithCustomDataGroup
}

func (pt pluginTemplate) IsPluginRule() bool {
	return pt.Plugin.Type == "PluginRule"
}

func (pt pluginTemplate) IsFieldActive(name string) bool {
	s := reflect.ValueOf(&pt.Plugin.Fields).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if typeOfT.Field(i).Name == name {
			v := f.Interface().(string)
			return v != ""
		}
	}
	return false
}

func (pt pluginTemplate) IsIntegerMutationRequired() bool {
	return pt.IsFieldActive("PluginID") || pt.IsFieldActive("PluginSID") ||
		pt.IsFieldActive("SrcPort") || pt.IsFieldActive("DstPort")
}
