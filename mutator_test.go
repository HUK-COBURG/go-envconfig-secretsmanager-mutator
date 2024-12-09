package envconfigsecret

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"testing"
)

func TestSecretsManagerMutator(t *testing.T) {
	suite.Run(t, new(MutatorTestSuite))
}

type MutatorTestSuite struct {
	suite.Suite
	localstackContainer  *localstack.LocalStackContainer
	mutator              *secretsManagerMutator
	secretsManagerClient *secretsmanager.Client
}

func (suite *MutatorTestSuite) SetupSuite() {
	var err error

	ctx := context.TODO()

	suite.localstackContainer, err = localstack.Run(ctx, "localstack/localstack:latest")
	if err != nil {
		suite.FailNow("create localstack testcontainer:", err)
		return
	}

	port, err := suite.localstackContainer.MappedPort(ctx, "4566/tcp")
	if err != nil {
		suite.FailNow("get localstack testcontainer port:", err)
		return
	}

	mutator := NewSecretsManagerMutator(ctx, func(o *secretsmanager.Options) {
		o.Region = "eu-central-1"
		o.BaseEndpoint = aws.String(fmt.Sprintf("http://localhost:%d", port.Int()))
	}).(*secretsManagerMutator)

	suite.mutator = mutator
	suite.secretsManagerClient = mutator.secretsManagerClient
}

func (suite *MutatorTestSuite) TestDoNotModifyNonSecretValues() {
	type config struct {
		NonSecretValue string `env:"MY_VAR"`
	}
	suite.T().Setenv("MY_VAR", "my-value")
	ctx := context.TODO()

	var cfg config
	err := envconfig.Process(ctx, &cfg, suite.mutator)
	suite.Require().NoError(err, "unexpected error processing config")

	suite.Equal(cfg.NonSecretValue, "my-value")
}

func (suite *MutatorTestSuite) TestGetValueFromSecretManager() {
	type config struct {
		SecretValue string `env:"SECRET_MY_VAR"`
	}
	ctx := context.TODO()

	secret, err := suite.secretsManagerClient.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String("MySecretVar"),
		SecretString: aws.String("superSecretValue"),
	})
	suite.Require().NoError(err, "unexpected error putting secret")

	suite.T().Setenv("SECRET_MY_VAR", "MySecretVar")
	var cfg1 config
	err = envconfig.Process(ctx, &cfg1, suite.mutator)
	if suite.NoError(err, "unexpected error processing config from name") {
		suite.Equal(cfg1.SecretValue, "superSecretValue")
	}

	suite.T().Setenv("SECRET_MY_VAR", *secret.ARN)
	var cfg2 config
	err = envconfig.Process(ctx, &cfg2, suite.mutator)
	if suite.NoError(err, "unexpected error processing config from ARN") {
		suite.Equal(cfg2.SecretValue, "superSecretValue")
	}
}

func (suite *MutatorTestSuite) TestErrorOnMissingSecret() {
	type config struct {
		SecretValue string `env:"SECRET_MY_VAR"`
	}
	ctx := context.TODO()

	suite.T().Setenv("SECRET_MY_VAR", "NonExistingSecret")
	var cfg1 config
	err := envconfig.Process(ctx, &cfg1, suite.mutator)
	suite.Require().Error(err, "unexpected error processing config")
	suite.Assert().ErrorContains(err, "get secret for key SECRET_MY_VAR:", "wrong error message")

}

func (suite *MutatorTestSuite) TearDownSuite() {
	err := suite.localstackContainer.Terminate(context.TODO())
	if err != nil {
		suite.T().Log("error terminating localstack testcontainer:", err)
	}
}
