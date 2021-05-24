module github.com/lovethedrake/brigdrake

go 1.15

replace github.com/mholt/caddy => github.com/caddyserver/caddy/v2 v2.3.0

require (
	github.com/brigadecore/brigade/sdk/v2 v2.0.0-alpha.3.0.20210430011302-da67f7eea600
	github.com/carolynvs/magex v0.5.0
	github.com/google/go-github/v33 v33.0.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lovethedrake/go-drake v0.15.0
	github.com/magefile/mage v1.11.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)
