package build

var githash string
var buildstamp string

func Version() (string, string) {
	return githash, buildstamp
}
