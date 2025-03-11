package utils

import (
	"testing"

	"github.com/assaidy/blogging_app/internal/utils"
)

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if len(hashedPassword) == 0 {
		t.Error("Hashed password is empty")
	}

	if hashedPassword == password {
		t.Error("Password wasn't hashed")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "mysecretpassword"

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !utils.VerifyPassword(password, hashedPassword) {
		t.Error("VerifyPassword failed: correct password did not match")
	}

	if utils.VerifyPassword("wrongpassword", hashedPassword) {
		t.Error("VerifyPassword failed: incorrect password matched")
	}
}

func TestVerifyPassword_EmptyInput(t *testing.T) {
	if utils.VerifyPassword("", "somehash") {
		t.Error("VerifyPassword failed: empty password should not match")
	}

	if utils.VerifyPassword("password", "") {
		t.Error("VerifyPassword failed: empty hash should not match")
	}
}
