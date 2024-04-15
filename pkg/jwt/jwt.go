package jwt

type JWTAPI interface {
	Account() string
	Role() EmproviseRole
}

//go:generate mockgen -destination mocks/jwt_mock.go -package mocks . JWTAPI
