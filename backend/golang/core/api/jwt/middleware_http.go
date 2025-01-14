package jwt

import (
	"context"
	"fmt"
	"net/http"
)

func Middleware(h http.HandlerFunc, secretJWT string, roleID ...uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		helper := NewHelper(secretJWT)

		cook, err := r.Cookie(AccessTokenName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("no cookie"))
			return
		}
		jwtToken := cook.Value

		tokenMC, err := helper.ParseToken(jwtToken)
		if err != nil {
			cook, err = r.Cookie(RefreshTokenName)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("bad access cookie. no refresh cookie"))
				return
			}
			refreshT := cook.Value

			mapClaims, err := helper.ParseToken(refreshT)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("bad access and refresh cookies"))
				return
			}
			claims := helper.ParseMapClaims(mapClaims)

			pair, err := helper.GeneratePair(claims.UserID, claims.IssuerName, claims.RoleID)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("bad access and refresh cookies"))
				return
			}

			accessCook, refreshCook := helper.PrepareCookies(pair)
			http.SetCookie(w, accessCook)
			http.SetCookie(w, refreshCook)

			jwtToken = pair.AccessToken
			tokenMC, err = helper.ParseToken(jwtToken)
		}

		tokenClaims := helper.ParseMapClaims(tokenMC)

		var RoleExist bool
		for _, rID := range roleID {
			if rID == tokenClaims.RoleID {
				RoleExist = true
				break
			}
		}

		if !RoleExist {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("forbidden"))
			return
		}

		ctx := context.WithValue(r.Context(), userIdCtxTag, tokenClaims.UserID)
		ctx = context.WithValue(ctx, roleIdCtxTag, tokenClaims.RoleID)
		h(w, r.WithContext(ctx))
	}
}

func unauthorized(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("unauthorized"))
}

func GetUserID(ctx context.Context) (string, error) {
	if mr, ok := ctx.Value(userIdCtxTag).(string); ok && mr != "" {
		return mr, nil
	}
	return "", fmt.Errorf("no user id in context")
}

func GetRoleID(ctx context.Context) (int, error) {
	mr := ctx.Value(roleIdCtxTag)
	switch roleId := mr.(type) {
	case int:
		return roleId, nil
	case int32:
		return int(roleId), nil
	case int64:
		return int(roleId), nil
	case uint:
		return int(roleId), nil
	case uint32:
		return int(roleId), nil
	case uint64:
		return int(roleId), nil
	}

	return 0, fmt.Errorf("something wrong with user role id in context")
}
