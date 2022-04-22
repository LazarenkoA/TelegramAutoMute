package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/xerrors"
	"os"
	"strings"
)

// noSignUp can be embedded to prevent signing up.
type noSignUp struct{}

func (c noSignUp) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, xerrors.New("not implemented")
}

func (c noSignUp) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

// termAuth implements authentication via terminal.
type termAuth struct {
	noSignUp
}

func (a termAuth) Phone(_ context.Context) (string, error) {
	var phone string
	fmt.Println("Введите номер телефона: ")
	fmt.Scanln(&phone)

	return phone, nil
}

func (a termAuth) Password(_ context.Context) (string, error) {
	var pwd string

	fmt.Println("Введите 2FA пароль: ")
	fmt.Scanln(&pwd)

	return strings.TrimSpace(pwd), nil
}

func (a termAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Введите код: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}
