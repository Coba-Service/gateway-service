package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	pb "github.com/MuhAndriJP/gateway-service.git/grpc/user"
	"github.com/MuhAndriJP/gateway-service.git/helper"
	"golang.org/x/oauth2"
	userInfo "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"

	"github.com/MuhAndriJP/gateway-service.git/entity"
	"github.com/MuhAndriJP/gateway-service.git/grpc/user/client"
	"github.com/labstack/echo/v4"
)

type Google struct {
	user client.Client
}

func (g *Google) HandleGoogleLogin(c echo.Context) (err error) {
	url := GoogleOauthConfig().AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	resp := &helper.Response{
		Code:    helper.SuccessCreated,
		Message: helper.StatusMessage[helper.SuccessCreated],
		Data: map[string]interface{}{
			"message": http.StatusTemporaryRedirect,
			"data":    url,
		},
	}

	return c.JSON(helper.HTTPStatusFromCode(helper.Success), resp)
	// return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (g *Google) HandleGoogleCallback(c echo.Context) (err error) {
	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	code := c.QueryParam("code")
	token, err := GoogleOauthConfig().Exchange(ctx, code)
	if err != nil {
		return
	}

	client := GoogleOauthConfig().Client(ctx, token)
	userInfo, err := client.Get(os.Getenv("GCP_USER_INFO"))
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}
	defer userInfo.Body.Close()

	user := &entity.Users{}
	if err = json.NewDecoder(userInfo.Body).Decode(&user); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if user.Email == "" {
		userInfo, err := getUserInfo(ctx, client)
		if err != nil {
			return err
		}

		req := pb.RegisterUserRequest{
			Name:  userInfo.Name,
			Email: userInfo.Email,
			Token: token.AccessToken,
		}
		bytes, _ := json.Marshal(user)
		_ = json.Unmarshal(bytes, &req)

		_, err = g.user.RegisterUser(ctx, &req)
		if err != nil {
			return err
		}

		resp := &helper.Response{
			Code:    helper.SuccessNoContent,
			Message: helper.StatusMessage[helper.SuccessNoContent],
			Data: map[string]interface{}{
				"token": token.AccessToken,
			},
		}

		return c.JSON(helper.HTTPStatusFromCode(helper.Success), resp)
	}

	resp := &helper.Response{
		Code:    helper.SuccessCreated,
		Message: helper.StatusMessage[helper.SuccessCreated],
		Data: map[string]interface{}{
			"token": token.AccessToken,
		},
	}

	return c.JSON(helper.HTTPStatusFromCode(helper.Success), resp)
}

func getUserInfo(ctx context.Context, client *http.Client) (*userInfo.Userinfo, error) {
	userService, err := userInfo.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	userInfo, err := userService.Userinfo.Get().Do()
	if err != nil {
		return nil, err
	}

	return userInfo, nil
}

func NewGoogleAuth() *Google {
	return &Google{}
}
