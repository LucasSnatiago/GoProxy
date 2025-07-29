package main

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func DisplayVersion() string {
	return "GoProxy version: " + version + "\n" +
		"Commit: " + commit + "\n" +
		"Date: " + date
}
