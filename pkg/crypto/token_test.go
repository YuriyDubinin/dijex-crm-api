package crypto

import "testing"

// ───────────────────────── Opaque-токен ─────────────────────────

func TestGenerateOpaqueToken_DefaultLength(t *testing.T) {
	tok, err := GenerateOpaqueToken(0) // 0 → DefaultTokenByteLength
	if err != nil {
		t.Fatalf("GenerateOpaqueToken: %v", err)
	}
	if tok == "" {
		t.Fatal("empty token")
	}
	// base64url без паддинга: 32 байта → 43 символа
	if want := 43; len(tok) != want {
		t.Errorf("len = %d, want %d", len(tok), want)
	}
}

func TestGenerateOpaqueToken_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		tok, err := GenerateOpaqueToken(16)
		if err != nil {
			t.Fatalf("GenerateOpaqueToken: %v", err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token generated: %s", tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	a := HashToken("hello")
	b := HashToken("hello")
	if a != b {
		t.Errorf("HashToken non-deterministic: %q vs %q", a, b)
	}
	// SHA-256 hex = 64 символа
	if len(a) != 64 {
		t.Errorf("len = %d, want 64", len(a))
	}
}

func TestHashToken_DifferentInputs(t *testing.T) {
	if HashToken("a") == HashToken("b") {
		t.Error("HashToken should differ for different inputs")
	}
}
