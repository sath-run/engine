package core

type UserInfo struct {
	Id    string
	Name  string
	Email string
}

func userLogin(username string, password string) error {
	// ctx := g.ContextWithToken(context.TODO())
	// res, err := g.grpcClient.Login(ctx, &pb.LoginRequest{
	// 	Account:  username,
	// 	Password: password,
	// })
	// if err != nil {
	// 	return err
	// }
	// g.userInfo = &UserInfo{
	// 	Name:  res.UserName,
	// 	Email: res.UserEmail,
	// 	Id:    res.UserId,
	// }
	// g.userToken = res.Token
	// if err := meta.SetCredentialUserToken(g.userToken); err != nil {
	// 	return err
	// }
	return nil
}

func Login(username string, password string) error {
	// err := userLogin(username, password)
	// if err != nil {
	// 	return err
	// }
	// g.heartbeatResetChan <- true
	return nil
}

func Logout() error {

	// // clear user token on DB
	// err := meta.SetCredentialUserToken("")
	// if err != nil {
	// 	return err
	// }

	// // clear user info in g
	// g.userToken = ""
	// g.userInfo = nil
	return nil
}

func GetUserInfo() *UserInfo {
	// return g.userInfo
	return nil
}
