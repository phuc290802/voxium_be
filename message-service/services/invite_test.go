package services

import (
	"strings"
	"testing"
)

func TestGenerateInviteCode(t *testing.T) {
	code1 := GenerateInviteCode()
	code2 := GenerateInviteCode()

	if len(code1) != 6 {
		t.Errorf("Expected code length 6, got %d", len(code1))
	}

	if code1 == code2 {
		t.Errorf("Generated codes should be different (randomness check)")
	}

	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for _, char := range code1 {
		if !strings.ContainsRune(charset, char) {
			t.Errorf("Code contains invalid character: %c", char)
		}
	}
}

func TestInviteLinkCycle(t *testing.T) {
	roomID := uint(123)
	code := "abcDEF"

	link := GenerateInviteLink(roomID, code)
	
	// Link should start with /join/
	if !strings.HasPrefix(link, "/join/") {
		t.Errorf("Link should start with /join/, got %s", link)
	}

	token := strings.TrimPrefix(link, "/join/")
	decodedRoomID, decodedCode, err := DecodeInviteLink(token)

	if err != nil {
		t.Fatalf("Failed to decode link: %v", err)
	}

	if decodedRoomID != roomID {
		t.Errorf("RoomID mismatch: expected %d, got %d", roomID, decodedRoomID)
	}

	if decodedCode != code {
		t.Errorf("Code mismatch: expected %s, got %s", code, decodedCode)
	}
}

func TestDecodeInvalidToken(t *testing.T) {
	_, _, err := DecodeInviteLink("invalid-token-!@#$")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}
