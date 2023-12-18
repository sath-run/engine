package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	pb "github.com/sath-run/engine/engine/core/protobuf"
	"github.com/sath-run/engine/utils"
)

type LoginCredential struct {
	UserId         string
	DeviceId       string
	Token          string
	Username       string
	Organization   string
	OrganizationId string
}

func Login(username string, password string, organization string) error {
	ctx := g.ContextWithToken(context.TODO())
	res, err := g.grpcClient.Login(ctx, &pb.LoginRequest{
		Account:      username,
		Password:     password,
		Organization: organization,
	})
	if err != nil {
		return err
	}
	credential := LoginCredential{
		Username:       username,
		Organization:   organization,
		UserId:         res.UserId,
		DeviceId:       res.DeviceId,
		Token:          res.Token,
		OrganizationId: res.OrganizationId,
	}
	if err := saveCredential(credential); err != nil {
		return err
	}
	g.heartbeatResetChan <- true
	return nil
}

func readCredential() *LoginCredential {
	dir := utils.ExecutableDir
	filename := filepath.Join(dir, ".sath.credential")
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	credential := LoginCredential{}
	if err := json.Unmarshal(bytes, &credential); err != nil {
		os.Remove(filename)
		return nil
	}
	return &credential
}

func saveCredential(credential LoginCredential) error {
	dir := utils.ExecutableDir
	data, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	g.credential = credential
	return os.WriteFile(filepath.Join(dir, ".sath.credential"), data, 0666)
}

func Logout() error {
	dir := utils.ExecutableDir
	if _, err := os.Stat(filepath.Join(dir, ".user.token")); !os.IsNotExist(err) {
		if err := os.Remove(filepath.Join(dir, ".user.token")); err != nil {
			return err
		}
	}

	// bytes, err := os.ReadFile(filepath.Join(dir, ".device.token"))
	// if err != nil {
	// 	return err
	// }
	// g.token = string(bytes)
	return nil
}

func Credential() LoginCredential {
	return g.credential
}
