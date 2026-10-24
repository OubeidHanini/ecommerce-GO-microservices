module github.com/OubeidHanini/ecommerce-GO-microservices/product-service

go 1.27.0

replace github.com/OubeidHanini/ecommerce-GO-microservices/platform => ../platform

require (
	github.com/OubeidHanini/ecommerce-GO-microservices/platform v0.0.0
	github.com/jackc/pgx/v5 v5.11.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/rabbitmq/amqp091-go v1.16.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
