package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HASH TESTS

const (
	testPassword = "cheetohDeadbolt123"
	altPassword  = "cheetohDeadbolt124"
)

func WasHashed(t *testing.T) {
	// passes if hashed password is indeed different from original password
	hashedPass, err := HashPassword(testPassword)
	if err != nil {
		t.Error(err)
	}
	if hashedPass == testPassword {
		t.Error("password was not hashed")
	}
}

func TestHashUnequal(t *testing.T) {
	// passes if CheckPasswordHash returns not nil as expected
	hashedPass, err := HashPassword(testPassword)
	if err != nil {
		t.Error(err)
	}
	match, _ := CheckPasswordHash(altPassword, hashedPass)
	if match {
		t.Error("password should not have matched, but did")
	}
}

func TestHashEqual(t *testing.T) {
	// passes if CheckPasswordHash returns nil as expected
	hashedPass, err := HashPassword(testPassword)
	if err != nil {
		t.Error(err)
	}
	match, _ := CheckPasswordHash(testPassword, hashedPass)
	if !match {
		t.Error("password should have matched, but did not")
	}
}

// JWT TESTS

func JWTRejectExpired(t *testing.T) {
	// passes if an expired JWT is properly rejected
	userID := uuid.New()
	tokenSecret := "very-secret-secret"
	expiration := time.Second * 2
	token, err := MakeJWT(userID, jwt.SigningMethodHS512, tokenSecret, expiration)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(2 * time.Second)
	_, err = ValidateJWT(token, "very-secret-secret", "HS256")
	if err == nil {
		t.Error("expired JWT not rejected")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	password1 := "correctPassword123!"
	password2 := "anotherPassword456!"
	hash1, _ := HashPassword(password1)
	hash1Plaintext := "$argon2id$v=19$m=65536,t=3,p=1$4XBycjFNyGzPuTk203FltA$Cw0x4cICG21uRv9zUHi+Gi7ygneSYO2+mmzc9a4EDSI"
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name          string
		password      string
		hash          string
		wantErr       bool
		matchPassword bool
	}{
		{
			name:          "Correct password",
			password:      password1,
			hash:          hash1,
			wantErr:       false,
			matchPassword: true,
		},
		{
			name:          "Incorrect password",
			password:      "wrongPassword",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Password doesn't match different hash",
			password:      password1,
			hash:          hash2,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Empty password",
			password:      "",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Invalid hash",
			password:      password1,
			hash:          "invalidhash",
			wantErr:       true,
			matchPassword: false,
		},
		{
			name:          "Password compares to pre-generated plaintext hash",
			password:      password1,
			hash:          hash1Plaintext,
			wantErr:       false,
			matchPassword: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && match != tt.matchPassword {
				t.Errorf("CheckPasswordHash() expects %v, got %v; here's hash: %v", tt.matchPassword, match, hash1)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validToken, _ := MakeJWT(userID, jwt.SigningMethodHS256, "secret", time.Hour)
	invalidToken, _ := MakeJWT(userID, jwt.SigningMethodHS384, "secret", time.Hour)

	tests := []struct {
		name        string
		tokenString string
		tokenSecret string
		wantUserID  uuid.UUID
		wantErr     bool
	}{
		{
			name:        "Valid token",
			tokenString: validToken,
			tokenSecret: "secret",
			wantUserID:  userID,
			wantErr:     false,
		},
		{
			name:        "Invalid token",
			tokenString: "invalid.token.string",
			tokenSecret: "secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "Wrong secret",
			tokenString: validToken,
			tokenSecret: "wrong_secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
		{
			name:        "Wrong algorithm",
			tokenString: invalidToken,
			tokenSecret: "wrong_secret",
			wantUserID:  uuid.Nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tt.tokenString, tt.tokenSecret, "HS256")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUserID != tt.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tt.wantUserID)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	const tokenWant = "thisIsATokenString"

	type testCases struct {
		name          string
		headers       http.Header
		expectedToken string
		expectErr     bool
	}

	cases := []testCases{
		{
			name:          "valid header",
			headers:       http.Header{"Authorization": []string{"Bearer " + tokenWant}},
			expectedToken: tokenWant,
			expectErr:     false,
		},
		{
			name:          "missing header",
			headers:       http.Header{},
			expectedToken: "",
			expectErr:     true,
		},
		{
			name:          "header present but empty",
			headers:       http.Header{"Authorization": []string{}},
			expectedToken: "",
			expectErr:     true,
		},
		{
			name:          "Bearer without token",
			headers:       http.Header{"Authorization": []string{"Bearer "}},
			expectedToken: "",
			expectErr:     true,
		},
		{
			name:          "incorrect scheme",
			headers:       http.Header{"Authorization": []string{"Token " + tokenWant}},
			expectedToken: "",
			expectErr:     true,
		},
		{
			name:          "no space after scheme",
			headers:       http.Header{"Authorization": []string{"Bearer" + tokenWant}},
			expectedToken: "",
			expectErr:     true,
		},
		{
			name:          "Different case Bearer",
			headers:       http.Header{"Authorization": []string{"bEaReR " + tokenWant}},
			expectedToken: tokenWant,
			expectErr:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token, err := GetBearerToken(c.headers)
			if (err != nil) != c.expectErr {
				t.Errorf("expected error: %v, got: %v", c.expectErr, err)
			}
			if token != c.expectedToken {
				t.Errorf("expected token: %v, got: %v", c.expectedToken, token)
			}
		})
	}
}
