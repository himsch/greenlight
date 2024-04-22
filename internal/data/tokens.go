package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"greenlight/internal/validator"
	"time"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
	ScopePasswordReset  = "password-reset"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	// 사용자 ID, 만료 및 범위 정보를 포함하는 토큰 인스턴스를 만듭니다.
	// 만료 시간을 얻기 위해 제공된 ttl(time-to-live) 기간 매개변수를 현재 시간에 추가한다는 점에 주목하세요.
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// 길이가 16바이트인 zero-value 바이트 슬라이스를 초기화합니다.
	randomBytes := make([]byte, 16)

	// crypto/rand 패키지의 Read() 함수를 사용하여
	// 운영 체제 CSPRNG의 임의 바이트(난수)로 바이트 슬라이스를 채웁니다.
	// CSPRNG가 올바르게 작동하지 않으면 오류가 반환됩니다.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// 바이트 조각을 base-32로 인코딩된 문자열로 인코딩하고 이를 토큰 일반 텍스트 필드에 할당합니다.
	// 이는 환영 이메일을 통해 사용자에게 보내는 토큰 문자열입니다.
	// 다음과 유사하게 보입니다.
	// Y3QMGX3PJ3WLRL2YRTQGQ6KRHU
	// 기본적으로 base-32 문자열의 끝에는 = 문자가 추가될 수 있습니다.
	// 토큰 목적으로는 이 패딩 문자가 필요하지 않으므로 아래 줄에서
	// WithPadding(base32.NoPadding) 메서드를 사용하여 이를 생략합니다.
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// 일반 텍스트 토큰 문자열의 SHA-256 해시를 생성합니다.
	// 이는 데이터베이스 테이블의 '해시' 필드에 저장하는 값입니다.
	// sha256.Sum256() 함수는 길이가 32인 *배열*을 반환하므로 작업을
	// 더 쉽게 하기 위해 저장하기 전에 [:] 연산자를 사용하여 슬라이스로 변환합니다.
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *sql.DB
}

func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m TokenModel) Insert(token *Token) error {
	query := `
    INSERT INTO tokens (hash, user_id, expiry, scope)
    VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
    DELETE FROM tokens
    WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userID)
	return err
}
