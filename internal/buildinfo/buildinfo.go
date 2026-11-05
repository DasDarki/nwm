package buildinfo

const Repository = "DasDarki/nwm"

var Version = "dev"

func IsDevelopment() bool {
	return Version == "dev" || Version == ""
}
