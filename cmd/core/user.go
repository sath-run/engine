package core

import (
	"context"

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

func Token() string {
	return g.token
}
