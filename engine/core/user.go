package core

import (
	"context"
	"sync"

	"github.com/sath-run/engine/constants"
	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/meta"
	"google.golang.org/grpc/metadata"
)

type User struct {
	mu          sync.RWMutex
	Id          string
	Name        string
	Email       string
	Token       string
	DeviceId    string
	DeviceToken string
}

func NewUser(grpc pb.EngineClient) (*User, error) {
	user := User{}
	deviceToken, err := meta.GetCredentialDeviceToken()
	if err == nil {
		user.DeviceToken = deviceToken
	} else if !constants.IsErrNil(err) {
		return nil, err
	}
	ctx := user.ContextWithToken(context.TODO())
	resp, err := grpc.HandShake(ctx, &pb.HandShakeRequest{
		SystemInfo: GetSystemInfo(),
	})
	if err != nil {
		return nil, err
	}
	user.DeviceToken = resp.Token
	user.DeviceId = resp.DeviceId

	if err := meta.SetCredentialDeviceToken(user.DeviceToken); err != nil {
		return nil, err
	}

	userToken, err := meta.GetCredentialUserToken()
	if err != nil && !constants.IsErrNil(err) {
		return nil, err
	}
	if len(userToken) > 0 {
		// refresh login data  usinguserToken
		user.Token = userToken
		user.Login(grpc, "", "")
	}

	return &user, nil
}

func (user *User) ContextWithToken(ctx context.Context) context.Context {
	var token string
	if tk := user.Token; tk != "" {
		token = tk
	} else {
		token = user.DeviceToken
	}
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", token,
		"version", constants.Version)
}

func (user *User) Login(grpc pb.EngineClient, username string, password string) error {
	res, err := grpc.Login(user.ContextWithToken(context.TODO()), &pb.LoginRequest{
		Account:  username,
		Password: password,
	})
	if err != nil {
		return err
	}
	user.mu.Lock()
	user.Token = res.Token
	user.Id = res.UserId
	user.Name = res.UserName
	user.Email = res.UserEmail
	user.mu.Unlock()
	if err := meta.SetCredentialUserToken(user.Token); err != nil {
		return err
	}
	return nil
}

func (user *User) Logout() error {
	user.mu.Lock()
	defer user.mu.Unlock()

	// clear user info
	user.Token = ""
	user.Id = ""
	user.Name = ""
	user.Email = ""

	// clear user token on DB
	err := meta.SetCredentialUserToken("")
	if err != nil {
		return err
	}

	return nil
}
