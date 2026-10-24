package postgres

import (
	"fmt"
	"net"
	"net/url"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/config"
)

type Config struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// ConfigFromEnv reads DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD(_FILE) and DB_SSLMODE.
func ConfigFromEnv(env *config.Env) Config {
	return Config{
		Host:     env.Required("DB_HOST"),
		Port:     env.Get("DB_PORT", "5432"),
		Name:     env.Required("DB_NAME"),
		User:     env.Required("DB_USER"),
		Password: env.Secret("DB_PASSWORD"),
		SSLMode:  env.Get("DB_SSLMODE", "prefer"),
	}
}

// url escapes every part: a generated password containing "@" or "/" would
// break a connection string built by hand.
func (c Config) url() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, c.Port),
		Path:     "/" + c.Name,
		RawQuery: url.Values{"sslmode": {c.SSLMode}}.Encode(),
	}
	return u.String()
}

// String keeps the password out of logs, even if the whole config is printed.
func (c Config) String() string {
	return fmt.Sprintf("postgres://%s:***@%s/%s?sslmode=%s", c.User, net.JoinHostPort(c.Host, c.Port), c.Name, c.SSLMode)
}
