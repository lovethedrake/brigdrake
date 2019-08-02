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
	podName string
}

func (t *timedOutError) Error() string {
	return fmt.Sprintf("timed out waiting for pod \"%s\" to complete", t.podName)
}
