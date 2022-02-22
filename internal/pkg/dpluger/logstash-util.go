package dpluger

import (
	"reflect"
	"strings"
)

func FieldMappingToLogstashField(fields *FieldMapping) {
	// iterate over fields to change them to logstash notation
	s := reflect.ValueOf(fields).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		// skip empty fields
		str := f.Interface().(string)
		if str == "" {
			continue
		}
		var v string
		if t := getType(str); t == ftES {
			// convert to logstash [field][subfield] notation
			v = LogstashFieldNotation(str)
			// do this except for timestamp, as it is only used in date filter
			if typeOfT.Field(i).Name != "Timestamp" {
				v = "%{" + v + "}"
			}
		} else {
			v = str
		}
		// set it
		setField(fields, typeOfT.Field(i).Name, v)
		// fmt.Printf("%d: %s %s = %v\n", i, typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
}

func LogstashFieldNotation(src string) (res string) {
	s := strings.Replace(src, "es:", "", 1)
	s = strings.Replace(s, "collect:", "", 1)
	s = strings.Replace(s, ".", "][", -1)
	s = strings.Replace(s, s, "["+s, 1)
	s = strings.Replace(s, s, s+"]", 1)
	res = s
	return
}
