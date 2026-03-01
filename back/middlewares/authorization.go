// package authorization to check client token api
package authorization

type Client struct {
	name string
}

var tokens map[string]Client

func InitAuthentication() {
	tokens = make(map[string]Client)
	tokens["0366b1ef49b042c9aa6b0950575b46e7d85014ec"] = Client{
		name: "admin",
	}
}

func VerifyToken(token string) (Client, bool) {
	value, res := tokens[token]

	return value, res
}
