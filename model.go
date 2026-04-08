package main

type BuildOptions struct {
	Fast         bool
	PullRequests map[string]string
	Releases     []string
}

func NewBuildOptions() *BuildOptions {
	return &BuildOptions{
		PullRequests: make(map[string]string),
	}
}
