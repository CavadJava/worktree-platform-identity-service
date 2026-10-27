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

func TestJWTManager_Generate_SetsUniqueJTI(t *testing.T) {
	m := NewJWTManager("test-secret", 60)

	token1, _, err := m.Generate("user-123", "admin")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	claims1, err := m.Verify(token1)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims1.JTI == "" {
		t.Fatal("expected a non-empty JTI claim")
	}

	token2, _, err := m.Generate("user-123", "admin")
	if err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}
	claims2, err := m.Verify(token2)
	if err != nil {
		t.Fatalf("second Verify failed: %v", err)
	}
	if claims2.JTI == "" {
		t.Fatal("expected a non-empty JTI claim on the second token")
	}
	if claims1.JTI == claims2.JTI {
		t.Fatal("expected two separate Generate calls to mint two different JTIs")
	}
}
