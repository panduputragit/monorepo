package token

import (
	"errors"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
)

var (
	ErrExpiredToken = errors.New("token has expired")
	ErrInvalidToken = errors.New("token is invalid")
)

type Payload struct {
	ID        string
	UserID    string
	Email     string
	Role      string
	IssuedAt  time.Time
	ExpiredAt time.Time
}

type Maker struct {
	key paseto.V4AsymmetricSecretKey
}

func NewMaker(hexKey string) (*Maker, error) {
	key, err := paseto.NewV4AsymmetricSecretKeyFromHex(hexKey)
	if err != nil {
		return nil, err
	}
	return &Maker{key: key}, nil
}

func NewMakerWithRandomKey() (*Maker, error) {
	key := paseto.NewV4AsymmetricSecretKey()
	return &Maker{key: key}, nil
}

func (m *Maker) CreateToken(userID, email, role string, duration time.Duration) (string, *Payload, error) {
	payload := &Payload{
		ID:        newTokenID(),
		UserID:    userID,
		Email:     email,
		Role:      role,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	token := paseto.NewToken()
	token.SetJti(payload.ID)
	token.SetSubject(payload.UserID)
	token.Set("email", payload.Email)
	token.Set("role", payload.Role)
	token.SetIssuedAt(payload.IssuedAt)
	token.SetExpiration(payload.ExpiredAt)

	signed := token.V4Sign(m.key, nil)
	return signed, payload, nil
}

func (m *Maker) VerifyToken(tokenStr string) (*Payload, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	token, err := parser.ParseV4Public(m.key.Public(), tokenStr, nil)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload := &Payload{}
	payload.ID, _ = token.GetJti()
	payload.UserID, _ = token.GetSubject()
	_ = token.Get("email", &payload.Email)
	_ = token.Get("role", &payload.Role)
	payload.IssuedAt, _ = token.GetIssuedAt()
	payload.ExpiredAt, _ = token.GetExpiration()

	return payload, nil
}

func newTokenID() string {
	key := paseto.NewV4AsymmetricSecretKey()
	return key.Public().ExportHex()[:32]
}
