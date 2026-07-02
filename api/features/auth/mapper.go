package auth

import "mrtutor/api/db/queries"

func PrincipalToCreateParam(principal Principal) queries.CreateUserParams {
	return queries.CreateUserParams{
		Username: principal.Username,
		Email:    principal.Email,
		Password: principal.HashedPassword,
	}
}

func ModelToPrincipal(user queries.User) Principal {
	return Principal{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		HashedPassword: user.Password,
		Role:           UserType(user.Role.String),
		CreateAt:       user.CreatedAt,
		ModifiedAt:     user.ModifiedAt,
	}
}
