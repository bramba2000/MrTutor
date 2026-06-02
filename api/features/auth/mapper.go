package auth

import "mrtutor/api/db/queries"

func PrincipalToCreateUserParam(principal Principal) queries.CreateUserParams {
	return queries.CreateUserParams{
		Username: principal.Username,
		Email:    principal.Email,
		Password: principal.HashedPassword,
	}
}

func UserToPrincipal(user queries.User) Principal {
	return Principal{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		HashedPassword: user.Password,
	}
}
