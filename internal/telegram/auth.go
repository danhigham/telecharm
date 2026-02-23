package telegram

import (
	"context"
	"errors"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// TUIAuth implements gotd's auth.UserAuthenticator using channels
// so the TUI can provide input asynchronously.
type TUIAuth struct {
	PhoneCh    chan string
	CodeCh     chan string
	PasswordCh chan string
	ErrCh      chan error

	// Callbacks to notify TUI what's needed
	OnPhoneRequested    func()
	OnCodeRequested     func()
	OnPasswordRequested func()
}

func NewTUIAuth() *TUIAuth {
	return &TUIAuth{
		PhoneCh:    make(chan string, 1),
		CodeCh:     make(chan string, 1),
		PasswordCh: make(chan string, 1),
		ErrCh:      make(chan error, 1),
	}
}

func (a *TUIAuth) Phone(ctx context.Context) (string, error) {
	if a.OnPhoneRequested != nil {
		a.OnPhoneRequested()
	}
	select {
	case phone := <-a.PhoneCh:
		return phone, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	if a.OnCodeRequested != nil {
		a.OnCodeRequested()
	}
	select {
	case code := <-a.CodeCh:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) Password(ctx context.Context) (string, error) {
	if a.OnPasswordRequested != nil {
		a.OnPasswordRequested()
	}
	select {
	case pw := <-a.PasswordCh:
		return pw, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (a *TUIAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (a *TUIAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("sign up not supported")
}
