// Package roleassign isolates the authorization decision for who may
// change whose role, expressed as a Strategy so the rule can be swapped
// or extended (e.g. a future cross-project superadmin) without touching
// the service layer that consumes it.
package roleassign

type Caller struct {
	UserID    string
	ProjectID string
	Role      string
}

type Target struct {
	UserID    string
	ProjectID string
}

type RoleAssigner interface {
	CanAssign(caller Caller, target Target) bool
}

// SameProjectAdmin implements the only rule the spec defines: an admin may
// change the role of any user within their own project, and nothing else.
type SameProjectAdmin struct{}

func NewSameProjectAdmin() *SameProjectAdmin {
	return &SameProjectAdmin{}
}

func (s *SameProjectAdmin) CanAssign(caller Caller, target Target) bool {
	return caller.Role == "admin" && caller.ProjectID == target.ProjectID
}
