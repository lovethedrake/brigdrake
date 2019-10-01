package drake

import "github.com/lovethedrake/brigdrake/pkg/brigade"

// Trigger is the public interface for all triggers.
type Trigger interface {
	Matches(brigade.Event) (bool, error)
	JobStatusNotifier(brigade.Event) (JobStatusNotifier, error)
}
