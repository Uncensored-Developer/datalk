package authenticator

import "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"

type JWTAuthenticator struct {
	usersAPI api.Client
}
