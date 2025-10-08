package oraws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awskms "github.com/aws/aws-sdk-go/service/kms"
)

// AWS implaments the AWS KMS client
type AWS struct {
	Svc   *awskms.KMS
	KeyID string
}

// NewAWS initializes a new AWS KMS client
func NewAWS(svc *awskms.KMS) *AWS {
	return &AWS{Svc: svc}
}

// CreateKey creates a new KMS key in AWS with metadata
func (a *AWS) CreateKey(keyname string) (string, error) {
	result, err := a.Svc.CreateKey(&awskms.CreateKeyInput{
		Tags: []*awskms.Tag{
			{
				TagKey:   aws.String("name"),
				TagValue: aws.String(keyname),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("could not create aws kms key: %w", err)
	}

	return *result.KeyMetadata.KeyId, nil
}

// EncryptData encrypts the `data` payload with the preconfigured
// AWS KMS key and returns the encrypted payload
func (a *AWS) EncryptData(data string, ctx context.Context) (*awskms.EncryptOutput, error) {
	result, err := a.Svc.EncryptWithContext(ctx, &awskms.EncryptInput{
		KeyId:     &a.KeyID,
		Plaintext: []byte(data),
	})
	if err != nil {
		return nil, fmt.Errorf("could not encrypt data: %w", err)
	}

	return result, nil
}

// DecryptData decrypts the `data` payload with AWS KMS key and retuns the
// decrypted payload
func (a *AWS) DecryptData(data string, ctx context.Context) (*awskms.DecryptOutput, error) {
	result, err := a.Svc.DecryptWithContext(ctx, &awskms.DecryptInput{
		CiphertextBlob: []byte(data),
	})
	if err != nil {
		return nil, fmt.Errorf("could not decrypt data: %w", err)
	}

	return result, nil
}

// CheckKeyExists checks if the provided KeyID exists in AWS
// returns error if the key is invalid or not found
func (a *AWS) CheckKeyExists() error {

	input := &awskms.DescribeKeyInput{
		KeyId: &a.KeyID,
	}

	_, err := a.Svc.DescribeKey(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("%s", aerr.Error())
		} else {
			return fmt.Errorf("could not check key: %w", err)
		}
	}

	return nil
}
