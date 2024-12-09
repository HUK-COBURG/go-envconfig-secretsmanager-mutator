# AWS Secrets Manager Config Mutator

This package is an extension for https://github.com/sethvargo/go-envconfig.
It implements a mutator to fetch secrets from the AWS Secrets Manager.

For every environment variable with the prefix `SECRET_`, the mutator assumes that the value configured as an
environment variable is the ARN or the Name of a secret in the Secrets Manager.
The mutator calls `GetSecretValue` with the provided value as the `SecretId` and replaces the original value with the
`SecretString` returned by the Secrets Manager.

---

## Usage

1. Create a secret in the AWS Secret Manager.
   In this example we assume that a secret named `FavoriteAnimal` is configured with the value `penguin`.
2. Configure an environment variable where the secret should be placed in with a key with the prefix `SECRET_` and a
   value containing either the name or the ARN of the secret.
```bash
SECRET_FAVORITE_ANIMAL=FavoriteAnimal
```

3. Add the variable to your application's config.

```go
type MyConfig struct{
	FavoriteAnimal string `env:"SECRET_FAVORITE_ANIMAL"`
	//additional values, also non-secret values possible
}

```

4. Process your config with the mutator configured.

```go
package main

import (
	"context"
	"log"

	"github.com/huk-coburg/go-envconfig-secretsmanager-mutator"
	"github.com/sethvargo/go-envconfig"
)

func main() {
	ctx := context.Background()

	var c MyConfig
	if err := envconfig.Process(ctx, &c, envconfigsecret.NewSecretsManagerMutator(ctx)); err != nil {
		log.Fatal(err)
	}

	// c.FavoriteAnimal = "penguin"
}
```

---

## Contributions
Feel free to open issues or submit pull requests to improve the package. Feedback and suggestions are always welcome!

---

