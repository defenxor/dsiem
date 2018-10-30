package dpluger

import "reflect"

type pluginTemplate struct {
	P          Plugin
	Creator    string
	CreateDate string
	R          tsvRef
	SIDField   string
}

func (pt pluginTemplate) IsPluginRule() bool {
	return pt.P.Type == "PluginRule"
}

func (pt pluginTemplate) IsFieldActive(name string) bool {
	s := reflect.ValueOf(&pt.P.Fields).Elem()
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
