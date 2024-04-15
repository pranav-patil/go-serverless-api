package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	JWTPrivateKey string = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAy1tFDaLVUABh4kcVWTBYm91oovV14Ex97mSJzS1BQwhZQ7ii
HUiNDJq8S82c+ccbVRg+Tj0H2N5802F0N5UZvKcYIxYd7Iejn+PNWQdQgyFKjea9
jULF32QvBZ988OA5qt3g3d0gq0ptU9Eh+yiSYpn8aYWr4F1nzTV+3pIhD6Lwe9JE
+7XrP2439KfGdEMdGhXxQVqXsb4hW8tIbsfmdxMrT+N6A7JB5KPRuiYcFRzImQq6
hPJYjSXS/8gAI5G2BPSGMA2fDOz6coFCtm3PnNsfO8MvDIdZU40EU2MRlVpxJeie
MU/+veXmnY/k3uL/gI0w5oL/Pav/0ZH3bbbmQQIDAQABAoIBADwwyaGdntHNKyvU
qmb5vmB0CnKhgMBhI60aKQeH65cxs2ouDh3oyYb/jdhKBbqQynBHermhqt7wC7Zt
U/7XrQR/2M8ZzsWh6DZ9MNy3I4eMpQqXT2euae8TMi/R4yNQ2wDOJ67DstXAc9ep
QQucGKgCuAlrvVHtk7nTqberPQnpQ2s5OWSQLgP2h1m0Wh2IFPRk/3Ak5IT5s146
rKibF0Chcb+SM9WwEQakUIRKNTYgp3nKN3znQLX5m7XsuXinUkNCPT0UM3UGXPaj
gitKdcz0+tmkwKAzAzzimTG7ljHTKH27X5hnAultJaCooiRW1L046wDmuU9C2JKw
y5xl4PkCgYEA7BPOzAMEX0S7glEN+CdNHKIWrpogFuxmw5zOTnf3CrNs2DqbD2bi
xrXsSHsMjQEhObhdcB0CmU8x1zDv7AZ8NBL6e1zsgszc2eNELjkOsvg0sMpHzIV0
6DIn+44TFqZYDTOZhziur72GrwNFNJdRwLMFkoEInt9omLoyAgtPnzsCgYEA3ISQ
W60of8G7S/qZ1aqNmvMFuq4jt0A0Y5SE1+y8iV1IYbE42f+ouYRSquQ2z4CmhTYe
sNrE4zOTaQ+VfkfHob4vJDLlWJ/nWQ6PLr3fDtH1icecxTQ5fJ0HTnKidsID6si9
S6hUgn3FdJpXUiuhEVnSJ1wVz4q8ZawQMMGWsLMCgYEAmxMf4q+QrawOqDnqPTpD
0y0+TQ99SNGdZ52Xf8AaDXNzak6FEQb6rKFQRwRdaDp3wtyytDS6Qk7dZIgG8joI
WISm+WY/DmTYJmC9psdgOnwE0KTvqQ95jhV0YjAfpd87M+DTVxoK1fJfiJNTYIqN
71EpteUA7qu+n6SfuOwJL4UCgYAl7KqDCcGoTyIuC/g+9ekKl/cJRv+feWxJH/bE
x9MY8LENFBSJ8V0MIsSw3TTL9P0udcNLeSRZSrp0XBjCsgeUOogS+qnU1xNLjqRz
TnY5L0TCIFFG3Rdx5fOmuzJTqERSMZnUlCuMkaLOzehsmlJGEKOC32Rk4CBMgA38
xJ5s3wKBgQCAkmHJKbo6WN3djyiHGtDSQQXK5aOQNhmOLhS7LalEe4uhi0hOD/Gy
of3uS6m2k7GYwnoCzpqrDA1vTeFIopMIcuZwGEbNL3md+2rmE5zLj4AspYq03izb
az1lfgEBRVtoaZQegNt2ZifGLyYmdfXRwgmS4VNSipv/iGPz8Psx/A==
-----END RSA PRIVATE KEY-----
`
	JWTTestIssuer  string = "test-issuer"
	JWTTestRoleFmt string = "urn:emprovise:identity:us-east-1:10:role/%s"
)

func pemToRsaPrivateKey(pemKey string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the key")
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsaKey, nil
}

func EmproviseJwtGenerator(issuer string, expired bool, accountID, role, pemKey string) string {
	var iat, exp *jwt.NumericDate

	iat = jwt.NewNumericDate(time.Now())

	if expired {
		exp = jwt.NewNumericDate(time.Now().Add(time.Hour * -1))
	} else {
		exp = jwt.NewNumericDate(time.Now().Add(time.Hour * 1))
	}

	claims := EmproviseClaims{
		jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  iat,
			ExpiresAt: exp,
		},
		Emprovise{
			Principal{
				Type:      "apikey",
				AccountID: accountID,
				Role:      role,
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	rasKey, _ := pemToRsaPrivateKey(pemKey)
	tokenString, _ := token.SignedString(rasKey)

	return tokenString
}
