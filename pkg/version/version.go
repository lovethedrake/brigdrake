package version

// Values for these are injected by the build
var (
	version string
	commit  string
)

// Version returns the BrigDrake version. This is typically a semantic version,
// but in the case of unreleased code, could be another descriptor such as
// "edge".
func Version() string {
	return version
}

// Commit returns the git commit SHA for the code that BrigDrake was built from.
func Commit() string {
	return commit
}
