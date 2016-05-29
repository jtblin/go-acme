/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package s3

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/jtblin/go-acme/backend"
	"github.com/jtblin/go-acme/types"
)

const (
	backendName      = "s3"
	awsBucketEnv     = "AWS_BUCKET"
	awsEncryptKeyEnv = "AWS_ENCRYPTION_KEY"
	awsEncryptAlgEnv = "AWS_ENCRYPTION_ALG"
	awsErrorNotFound = "NoSuchKey"
	awsRegionEnv     = "AWS_REGION"
	storageFilename  = "cert.json"
)

type storage struct {
	bucket              string
	encryptionAlgorithm string
	encryptionKey       string
	s3                  S3
	storageLock         sync.RWMutex
}

type awsSDKProvider struct {
	creds *credentials.Credentials
}

// Services is an abstraction over AWS, to allow mocking/other implementations.
type Services interface {
	Metadata() (EC2Metadata, error)
	Storage(region string) (S3, error)
}

// EC2Metadata is an abstraction over the AWS metadata service.
type EC2Metadata interface {
	// Query the EC2 metadata service (used to discover instance-id etc).
	GetMetadata(path string) (string, error)
}

// S3 is an abstraction over S3, to allow mocking/other implementations.
// Note that the ListX functions return a list, so callers don't need to deal with paging.
type S3 interface {
	// Get an object from S3.
	GetObject(request *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	// Put an object in S3.
	PutObject(request *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// awsSdkS3 is an implementation of the S3 interface, backed by aws-sdk-go.
type awsSdkS3 struct {
	s3 *s3.S3
}

// GetObject gets an object from an s3 bucket.
func (s *awsSdkS3) GetObject(request *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	response, err := s.s3.GetObject(request)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == awsErrorNotFound {
				return nil, nil
			}
		}

		return nil, err
	}
	return response, nil
}

// PutObject puts an object in an s3 bucket.
func (s *awsSdkS3) PutObject(request *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return s.s3.PutObject(request)
}

// Metadata is an implementation of EC2 Metadata.
func (p *awsSDKProvider) Metadata() (EC2Metadata, error) {
	client := ec2metadata.New(session.New(&aws.Config{}))
	return client, nil
}

// Storage is an implementation of S3 Storage.
func (p *awsSDKProvider) Storage(regionName string) (S3, error) {
	service := s3.New(session.New(&aws.Config{
		Region:      &regionName,
		Credentials: p.creds,
	}))

	s3 := &awsSdkS3{
		s3: service,
	}
	return s3, nil
}

// Name returns the display name of the backend.
func (s *storage) Name() string {
	return backendName
}

func key(domain string) string {
	parts := strings.Split(domain, ".")
	keySlice := make([]string, len(parts))
	l := len(parts) - 1
	for idx, part := range parts {
		keySlice[l-idx] = part
	}
	keySlice = append(keySlice, storageFilename)
	return strings.Join(keySlice, "/")
}

// SaveAccount saves the account to s3.
func (s *storage) SaveAccount(account *types.Account) error {
	s.storageLock.Lock()
	defer s.storageLock.Unlock()

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}

	req := &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key(account.DomainsCertificate.Domain.Main)),
	}
	if s.encryptionAlgorithm != "" && s.encryptionKey != "" {
		req.SSECustomerAlgorithm = aws.String(s.encryptionAlgorithm)
		req.SSECustomerKey = aws.String(s.encryptionKey)
		req.SSECustomerKeyMD5 = aws.String(fmt.Sprintf("%x", md5.Sum([]byte(s.encryptionKey))))
	}
	_, err = s.s3.PutObject(req)
	return err
}

// LoadAccount loads the account from s3.
func (s *storage) LoadAccount(domain string) (*types.Account, error) {
	s.storageLock.RLock()
	defer s.storageLock.RUnlock()

	req := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key(domain)),
	}
	if s.encryptionAlgorithm != "" && s.encryptionKey != "" {
		req.SSECustomerAlgorithm = aws.String(s.encryptionAlgorithm)
		req.SSECustomerKey = aws.String(s.encryptionKey)
		req.SSECustomerKeyMD5 = aws.String(fmt.Sprintf("%x", md5.Sum([]byte(s.encryptionKey))))
	}

	resp, err := s.s3.GetObject(req)
	if err != nil || resp == nil {
		return nil, err
	}

	defer resp.Body.Close()
	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	account := types.Account{
		DomainsCertificate: &types.DomainCertificate{},
	}
	if err := json.Unmarshal(file, &account); err != nil {
		return nil, fmt.Errorf("Error loading account: %v", err)
	}

	return &account, nil
}

func newBackend(awsServices Services) (backend.Interface, error) {
	bucket := os.Getenv(awsBucketEnv)
	if bucket == "" {
		return nil, errors.New("missing bucket name")
	}
	region := os.Getenv(awsRegionEnv)
	if region == "" {
		metadata, err := awsServices.Metadata()
		if err != nil {
			return nil, fmt.Errorf("error creating AWS metadata client: %v", err)
		}

		var document struct{ region string }
		doc, err := metadata.GetMetadata("dynamic/instance-identity/document")
		if err != nil {
			return nil, fmt.Errorf("error getting region: %v", err)
		}
		if err = json.Unmarshal([]byte(doc), &document); err != nil {
			return nil, fmt.Errorf("error parsing region: %v", err)
		}
		region = document.region
	}
	s3, err := awsServices.Storage(region)
	if err != nil {
		return nil, err
	}
	return &storage{
		bucket:              bucket,
		encryptionAlgorithm: os.Getenv(awsEncryptAlgEnv),
		encryptionKey:       os.Getenv(awsEncryptKeyEnv),
		s3:                  s3,
	}, nil
}

func newAWSSDKProvider(creds *credentials.Credentials) *awsSDKProvider {
	return &awsSDKProvider{creds: creds}
}

func init() {
	backend.RegisterBackend(backendName, func() (backend.Interface, error) {
		creds := credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&ec2rolecreds.EC2RoleProvider{
					Client: ec2metadata.New(session.New(&aws.Config{})),
				},
				&credentials.SharedCredentialsProvider{},
			})
		aws := newAWSSDKProvider(creds)
		return newBackend(aws)
	})
}
