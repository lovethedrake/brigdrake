package brigade

import "fmt"

type multiError struct {
	errs []error
}

func (m *multiError) Error() string {
	str := fmt.Sprintf("%d errors encountered: ", len(m.errs))
	for i, err := range m.errs {
		str = fmt.Sprintf("%s\n%d. %s", str, i, err.Error())
	}
	return str
}

type timedOutError struct {
	job string
}

func (t *timedOutError) Error() string {
	return fmt.Sprintf("timed out waiting for job %q to complete", t.job)
}

type errPendingJobCanceled struct {
	job string
}

func (e *errPendingJobCanceled) Error() string {
	return fmt.Sprintf("pending job %q canceled", e.job)
}

type errInProgressJobAborted struct {
	job string
}

func (e *errInProgressJobAborted) Error() string {
	return fmt.Sprintf("in-progress job %q aborted", e.job)
}
