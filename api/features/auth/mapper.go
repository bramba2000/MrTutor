package auth

import "mrtutor/api/db/queries"

func PrincipalToCreateParam(principal Principal) queries.CreatePrincipalParams {
	return queries.CreatePrincipalParams{
		Username: principal.Username,
		Email:    principal.Email,
		Password: principal.HashedPassword,
		Role:     string(principal.Role),
	}
}

func ModelToPrincipal(user queries.User) Principal {
	return Principal{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		HashedPassword: user.Password,
		Role:           UserRole(user.Role),
		CreateAt:       user.CreatedAt,
		ModifiedAt:     user.ModifiedAt,
	}
}
