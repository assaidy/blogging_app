package utils

import (
	"testing"
)

type TestUser struct {
	Username string `validate:"required,customUsername"`
	FullName string `validate:"required,customNoOuterSpaces"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name        string
		input       TestUser
		expectedErr string
	}{
		{
			name: "Valid Input",
			input: TestUser{
				Username: "valid_user123",
				FullName: "John Doe",
			},
			expectedErr: "",
		},
		{
			name: "Invalid Username - Special Characters",
			input: TestUser{
				Username: "invalid-user!",
				FullName: "John Doe",
			},
			expectedErr: "Username: violation in constraint 'customUsername'",
		},
		{
			name: "Invalid Username - Empty",
			input: TestUser{
				Username: "",
				FullName: "John Doe",
			},
			expectedErr: "Username: violation in constraint 'required'",
		},
		{
			name: "Invalid FullName - Leading Space",
			input: TestUser{
				Username: "valid_user123",
				FullName: " John Doe",
			},
			expectedErr: "FullName: violation in constraint 'customNoOuterSpaces'",
		},
		{
			name: "Invalid FullName - Trailing Space",
			input: TestUser{
				Username: "valid_user123",
				FullName: "John Doe ",
			},
			expectedErr: "FullName: violation in constraint 'customNoOuterSpaces'",
		},
		{
			name: "Invalid FullName - Empty",
			input: TestUser{
				Username: "valid_user123",
				FullName: "",
			},
			expectedErr: "FullName: violation in constraint 'required'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if err != nil {
				if tt.expectedErr == "" {
					t.Errorf("Unexpected error: %v", err)
				} else if err.Error() != tt.expectedErr {
					t.Errorf("Expected error: %v, got: %v", tt.expectedErr, err)
				}
			} else if tt.expectedErr != "" {
				t.Errorf("Expected error: %v, got nil", tt.expectedErr)
			}
		})
	}
}
