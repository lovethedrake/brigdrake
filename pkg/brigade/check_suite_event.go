package brigade

type checkSuiteEvent struct {
	Body checkSuiteBody `json:"body"`
}

type checkSuiteBody struct {
	CheckSuite checkSuite `json:"check_suite"`
}

type checkSuite struct {
	HeadBranch *string `json:"head_branch"`
}
