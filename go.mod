module github.com/eurofurence/reg-payment-service

go 1.18

require (
	github.com/StephanHCB/go-autumn-logging v0.3.0
	github.com/StephanHCB/go-autumn-logging-zerolog v0.5.0
	github.com/StephanHCB/go-autumn-restclient v0.8.0
	github.com/StephanHCB/go-autumn-restclient-circuitbreaker v0.4.1
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-http-utils/headers v0.0.0-20181008091004-fed159eddc2a
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/google/uuid v1.6.0
	gorm.io/driver/mysql v1.5.5
	gorm.io/gorm v1.25.7
)

require github.com/rs/zerolog v1.32.0

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/stretchr/testify v1.9.0
	gopkg.in/yaml.v3 v3.0.1
)
