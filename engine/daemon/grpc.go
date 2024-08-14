package daemon

import (
	"context"
	"crypto/tls"

	"github.com/sath-run/engine/constants"
	pb "github.com/sath-run/engine/engine/daemon/protobuf"
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
	Id    string
	Name  string
	Email string
	Token string
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

	ctx, _ := c.AppendToOutgoingContext(context.TODO())

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

	if userToken, err := meta.GetCredentialUserToken(); userToken == "" && err == nil {
		// refresh login data  using userToken
		ctx := metadata.AppendToOutgoingContext(ctx,
			"authorization", userToken)

		// we can safely ignore login error
		c.Login(ctx, "", "")
	} else if err != nil && !constants.IsErrNil(err) {
		return nil, err
	}

	return &c, nil
}

func (c *Connection) AppendToOutgoingContext(ctx context.Context) (context.Context, bool) {
	var token string
	var hasUser bool
	if u := c.user; u != nil {
		token = u.Token
		hasUser = true
	} else {
		token = c.deviceToken
		hasUser = false
	}
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", token,
		"version", constants.Version), hasUser
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
		Token: res.Token,
		Id:    res.UserId,
		Name:  res.UserName,
		Email: res.UserEmail,
	}
	if err := meta.SetCredentialUserToken(user.Token); err != nil {
		return err
	}
	c.user = &user
	return nil
}

func (c *Connection) Logout() error {
	// clear user token on DB
	err := meta.SetCredentialUserToken("")
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
