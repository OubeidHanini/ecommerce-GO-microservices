package rabbitmq

import (
	"fmt"
	"net"
	"net/url"

	"github.com/OubeidHanini/ecommerce-GO-microservices/platform/config"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
}

// ConfigFromEnv reads MQ_HOST, MQ_PORT, MQ_USER and MQ_PASSWORD(_FILE).
func ConfigFromEnv(env *config.Env) Config {
	return Config{
		Host:     env.Required("MQ_HOST"),
		Port:     env.Get("MQ_PORT", "5672"),
		User:     env.Required("MQ_USER"),
		Password: env.Secret("MQ_PASSWORD"),
	}
}

// url escapes the credentials, which may contain URL-reserved characters.
func (c Config) url() string {
	u := url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   "/",
	}
	return u.String()
}

// String keeps the password out of logs, even if the whole config is printed.
func (c Config) String() string {
	return fmt.Sprintf("amqp://%s:***@%s/", c.User, net.JoinHostPort(c.Host, c.Port))
}
