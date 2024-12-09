package envconfigsecret

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sethvargo/go-envconfig"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"strings"
)

type secretsManagerMutator struct {
	secretsManagerClient *secretsmanager.Client
}

// NewSecretsManagerMutator returns a new envconfig.Mutator that retrieves secrets from the AWS Secrets Manager.
// In order to configure a secret, added an env variable with a key starting with "SECRET_".
// The value of the secret is the secret id (name or ARN) of the secret to load from secrets manager.
// The mutator replaces the secret id with the value received from the Secrets Manager.
func NewSecretsManagerMutator(ctx context.Context, opts ...func(o *secretsmanager.Options)) envconfig.Mutator {
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		panic("failed to load default aws config: " + err.Error())
	}
	otelaws.AppendMiddlewares(&awsCfg.APIOptions)

	return &secretsManagerMutator{
		secretsManagerClient: secretsmanager.NewFromConfig(awsCfg, opts...),
	}
}

// EnvMutate to implement the envconfig.Mutator interface.
func (mutator *secretsManagerMutator) EnvMutate(ctx context.Context, _, key, _, originalValue string) (resultValue string, stop bool, err error) {
	// If the value is no secret, return without modifications
	if !strings.HasPrefix(key, "SECRET_") {
		return originalValue, false, nil
	}

	defer func() {
		// If there is any error, add info which key was loaded
		if err != nil {
			err = fmt.Errorf("get secret for key %s: %w", key, err)
		}
	}()

	// Assume that the original value is the secret id for secret manager, so get the value from the secret manager
	secretValue, err := mutator.secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(originalValue),
	})
	if err != nil {
		return originalValue, false, fmt.Errorf("get value from secret manager: %w", err)
	}

	// Only secret strings are supported
	if secretValue.SecretString == nil {
		return originalValue, false, fmt.Errorf("no secret string value found")
	}

	return *secretValue.SecretString, false, nil
}
