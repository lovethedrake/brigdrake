package brigade

// TODO: Things in this file should move into a github-specific VCS package.

type checkSuiteEvent struct {
	Body checkSuiteBody `json:"body"`
}

type checkSuiteBody struct {
	CheckSuite checkSuite `json:"check_suite"`
}

type checkSuite struct {
	HeadBranch *string `json:"head_branch"`
}
