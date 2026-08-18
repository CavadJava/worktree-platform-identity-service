package roleassign

import "testing"

func TestSameProjectAdmin_CanAssign(t *testing.T) {
	strategy := NewSameProjectAdmin()

	tests := []struct {
		name   string
		caller Caller
		target Target
		want   bool
	}{
		{
			name:   "admin in same project can assign",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "admin"},
			target: Target{UserID: "u2", ProjectID: "p1"},
			want:   true,
		},
		{
			name:   "admin in different project cannot assign",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "admin"},
			target: Target{UserID: "u2", ProjectID: "p2"},
			want:   false,
		},
		{
			name:   "non-admin caller cannot assign",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "user"},
			target: Target{UserID: "u2", ProjectID: "p1"},
			want:   false,
		},
		{
			name:   "admin with empty project id cannot assign to target with empty project id",
			caller: Caller{UserID: "u1", ProjectID: "", Role: "admin"},
			target: Target{UserID: "u2", ProjectID: ""},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.CanAssign(tt.caller, tt.target)
			if got != tt.want {
				t.Errorf("CanAssign(%+v, %+v) = %v, want %v", tt.caller, tt.target, got, tt.want)
			}
		})
	}
}

func TestSuperadminOrSameProjectAdmin_CanAssign(t *testing.T) {
	strategy := NewSuperadminOrSameProjectAdmin()

	tests := []struct {
		name   string
		caller Caller
		target Target
		want   bool
	}{
		{
			name:   "superadmin can assign across any project",
			caller: Caller{UserID: "u1", ProjectID: "", Role: "superadmin"},
			target: Target{UserID: "u2", ProjectID: "p1"},
			want:   true,
		},
		{
			name:   "same-project admin can still assign (delegates to SameProjectAdmin rule)",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "admin"},
			target: Target{UserID: "u2", ProjectID: "p1"},
			want:   true,
		},
		{
			name:   "cross-project admin still cannot assign",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "admin"},
			target: Target{UserID: "u2", ProjectID: "p2"},
			want:   false,
		},
		{
			name:   "plain user cannot assign",
			caller: Caller{UserID: "u1", ProjectID: "p1", Role: "user"},
			target: Target{UserID: "u2", ProjectID: "p1"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.CanAssign(tt.caller, tt.target)
			if got != tt.want {
				t.Errorf("CanAssign(%+v, %+v) = %v, want %v", tt.caller, tt.target, got, tt.want)
			}
		})
	}
}
