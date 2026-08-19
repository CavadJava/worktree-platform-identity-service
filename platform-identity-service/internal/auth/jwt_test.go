package auth

import "testing"

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	m := NewJWTManager("test-secret", 60)

	token, _, err := m.Generate("user-123", "admin")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.UserID != "user-123" || claims.SystemRole != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestJWTManager_Verify_RejectsWrongSecret(t *testing.T) {
	m1 := NewJWTManager("secret-one", 60)
	m2 := NewJWTManager("secret-two", 60)

	token, _, err := m1.Generate("user-123", "user")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if _, err := m2.Verify(token); err == nil {
		t.Error("expected Verify to fail with mismatched secret")
	}
}
