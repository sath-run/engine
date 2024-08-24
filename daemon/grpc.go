package daemon

import (
	"context"
	"crypto/tls"

	"github.com/rs/zerolog/log"
	"github.com/sath-run/engine/constants"
	pb "github.com/sath-run/engine/daemon/protobuf"
	"github.com/sath-run/engine/meta"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Connection struct {
	pb.EngineClient
	deviceId    string
	deviceToken string
	user        *User
}

type User struct {
	token string
	Id    string
	Name  string
	Email string
}

func NewConnection(address string, ssl bool) (*Connection, error) {

	var credential credentials.TransportCredentials
	if ssl {
		credential = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
		})
	} else {
		credential = insecure.NewCredentials()
	}
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(credential))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	client := pb.NewEngineClient(conn)
	return NewConnectionWithClient(client)
}

func NewConnectionWithClient(client pb.EngineClient) (*Connection, error) {
	c := Connection{
		EngineClient: client,
	}
	if deviceToken, err := meta.GetCredentialDeviceToken(); err == nil {
		c.deviceToken = deviceToken
	} else if !constants.IsErrNil(err) {
		return nil, err
	}

	ctx := c.AppendToOutgoingContext(context.TODO(), nil)

	// get or refresh device token from server
	resp, err := c.HandShake(ctx, &pb.HandShakeRequest{
		SystemInfo: GetSystemInfo(),
	})
	if err != nil {
		return nil, err
	}
	c.deviceToken = resp.Token
	c.deviceId = resp.DeviceId

	if err := meta.SetCredentialDeviceToken(c.deviceToken); err != nil {
		return nil, err
	}

	if userToken, err := meta.GetCredentialUserToken(); userToken != "" && err == nil {
		// refresh login data using userToken
		ctx := c.AppendToOutgoingContext(ctx, &User{token: userToken})
		if err := c.Login(ctx, "", ""); err != nil {
			// we can safely ignore login error
			log.Warn().Err(err).Msg("error login user")
		}
	} else if err != nil && !constants.IsErrNil(err) {
		return nil, err
	}
	return &c, nil
}

func (c *Connection) AppendToOutgoingContext(ctx context.Context, user *User, kv ...string) context.Context {
	kv = append(kv, "version", constants.Version)
	if user != nil {
		kv = append(kv, "authorization", user.token)
	} else if u := c.user; u != nil {
		kv = append(kv, "authorization", u.token)
	} else {
		kv = append(kv, "authorization", c.deviceToken)
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}

func (c *Connection) Login(ctx context.Context, username string, password string) error {
	res, err := c.EngineClient.Login(ctx, &pb.LoginRequest{
		Account:  username,
		Password: password,
	})
	if err != nil {
		return err
	}
	user := User{
		token: res.Token,
		Id:    res.UserId,
		Name:  res.UserName,
		Email: res.UserEmail,
	}
	if err := meta.SetCredentialUserToken(user.Id); err != nil {
		return err
	}
	c.user = &user
	return nil
}

func (c *Connection) Logout() error {
	// clear user token on DB
	err := meta.RemoveCredentialUserToken()
	if err != nil {
		return err
	}
	c.user = nil
	return nil
}

func (c *Connection) User() *User {
	if c.user == nil {
		return nil
	}
	// copy g.user to a new struct
	user := *c.user
	return &user
}
