module github.com/hightrex/microservices-platform/tests/security/tenant-isolation

go 1.25.7

require (
	github.com/google/uuid v1.6.0
	github.com/hightrex/microservices-platform/libs/go v0.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hightrex/microservices-platform/libs/go => ../../../libs/go
