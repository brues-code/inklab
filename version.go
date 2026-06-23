package main

// Build-time version metadata, injected via -ldflags at release time:
//
//	-X main.Version=<git tag> -X main.Repo=<owner/repo>
//
// (see .github/workflows/release.yml). The defaults keep the update check
// disabled for local/dev builds, where there is no meaningful version to
// compare against.
var (
	Version = "dev"
	Repo    = ""
)
