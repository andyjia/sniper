package hook

import (
	"context"
	"fmt"
	"sniper/util/auth"
	"sniper/util/errors"
	"sniper/util/mc"
	"strings"

	"github.com/bilibili/twirp"
)

// NewCheckLogin 检查用户登录态，未登录直接报错返回
func NewCheckLogin() *twirp.ServerHooks {
	return &twirp.ServerHooks{
		RequestRouted: func(ctx context.Context) (context.Context, error) {
			// example sniper provided.
			// if ctxkit.GetUserID(ctx) == 0 {
			// 	return ctx, errors.NotLoginError
			// }
			// return ctx, nil

			// individual authentication method
			// req, _ := twirp.Request(ctx)
			// tokenString := strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", -1)
			// fmt.Printf("tokenString=%s\n", tokenString)
			// if tokenString == "" {
			// 	return ctx, errors.NotLoginError
			// }
			// token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 	// Don't forget to validate the alg is what you expect:
			// 	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			// 		return false, errors.NotLoginError
			// 	}

			// 	// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			// 	return []byte("test"), nil
			// })
			// fmt.Printf("token=%v\n", token)
			// if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 	fmt.Printf("ok=%v\n", claims)
			// 	return ctx, nil
			// }
			// fmt.Println(err)
			// return ctx, errors.NotLoginError

			req, _ := twirp.Request(ctx)
			tokenString := strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", -1)
			fmt.Printf("tokenString=%s\n", tokenString)
			if tokenString == "" {
				return ctx, errors.NotLoginError
			}
			_, err := auth.Authenticate(tokenString)
			if err != nil {
				return ctx, errors.NotLoginError
			}

			c := mc.Get(ctx, "DEFAULT")
			itm, err := c.Get(ctx, "someone")
			if err != nil {
				return ctx, errors.NotLoginError
			}
			if string(itm.Value) != tokenString {
				return ctx, errors.NotLoginError
			}

			return ctx, nil

		},
	}
}
