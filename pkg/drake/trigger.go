package drake

import "github.com/lovethedrake/canard/pkg/brigade"

// Trigger is the public interface for all triggers.
type Trigger interface {
	Matches(brigade.Event) (bool, error)
}
