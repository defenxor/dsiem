// TODO consolidate all APM related stuff here

package apm

var enabled bool

//Enabled returns whether apm is enabled
func Enabled() bool {
	return enabled
}

//Enable set apm status
func Enable(e bool) {
	enabled = e
}
