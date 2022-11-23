package core

import (
	"context"
	"os"
	"path/filepath"

	pb "github.com/sath-run/engine/pkg/protobuf"
)

func Login(email string, password string) error {
	ctx := g.ContextWithToken(context.Background())
	res, err := g.grpcClient.Login(ctx, &pb.LoginRequest{
		Account:  email,
		Password: password,
	})
	if err != nil {
		return err
	}
	if err := saveToken(res.Token, true); err != nil {
		return err
	}
	g.token = res.Token
	return nil
}

func Logout() error {
	dir, err := getExecutableDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".user.token")); !os.IsNotExist(err) {
		if err := os.Remove(filepath.Join(dir, ".user.token")); err != nil {
			return err
		}
	}

	bytes, err := os.ReadFile(filepath.Join(dir, ".device.token"))
	if err != nil {
		return err
	}
	g.token = string(bytes)
	return nil
}

func Token() string {
	return g.token
}
