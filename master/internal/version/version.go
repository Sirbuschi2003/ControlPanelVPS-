package version

// Set at build time via:
// -X 'github.com/Sirbuschi2003/ControlPanelVPS/master/internal/version.Commit=<sha>'
// -X 'github.com/Sirbuschi2003/ControlPanelVPS/master/internal/version.Date=<rfc3339>'
var (
	Commit = "dev"
	Date   = "unknown"
)
