package main

import (
	"io"
	"os"
	"sort"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

const (
	endpointKey = "endpoint"
)

type ObjectStore struct {
	log    logrus.FieldLogger
	client *obs.ObsClient
	bucket string
}

func newObjectStore(logger logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: logger}
}

func (o *ObjectStore) Init(config map[string]string) error {
	if err := veleroplugin.ValidateObjectStoreConfigKeys(config, endpointKey); err != nil {
		return err
	}
	endpoint := config[endpointKey]
	if err := loadEnv(); err != nil {
		return err
	}
	accessKey := os.Getenv("OBS_ACCESS_KEY")
	secretKey := os.Getenv("OBS_SECRET_KEY")
	err := validate(endpoint, accessKey, secretKey)
	if err != nil {
		return err
	}
	client, err := obs.New(accessKey, secretKey, endpoint)
	if err != nil {
		return err
	}
	o.client = client
	return nil
}

func loadEnv() error {
	envFile := os.Getenv("HUAWEI_CLOUD_CREDENTIALS_FILE")
	if envFile == "" {
		return errors.New("credentials file not found")
	}

	if err := godotenv.Overload(envFile); err != nil {
		return errors.Wrapf(err, "error loading environment from HUAWEI_CLOUD_CREDENTIALS_FILE (%s)", envFile)
	}

	return nil
}

func validate(endpoint, accessKey, secretKey string) error {
	if endpoint == "" {
		return errors.New("no obs endpoint in config file")
	}

	if accessKey == "" && secretKey != "" {
		return errors.New("no obs access_key specified")
	}

	if accessKey != "" && secretKey == "" {
		return errors.New("no obs secret_key specified")
	}

	if accessKey == "" && secretKey == "" {
		return errors.New("no obs secret_key and access_key specified")
	}
	return nil
}

func (o *ObjectStore) PutObject(bucket, key string, body io.Reader) error {
	input := &obs.PutObjectInput{}
	input.Bucket = bucket
	input.Key = key
	input.Body = body
	_, err := o.client.PutObject(input)
	if err != nil {
		return errors.Wrapf(err, "failed to put object %s", key)
	}
	return nil
}

func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {
	log := o.log.WithFields(
		logrus.Fields{
			"bucket": bucket,
			"key":    key,
		},
	)
	log.Debug("Checking if object exists")
	input := &obs.GetObjectMetadataInput{Bucket: bucket, Key: key}
	_, err := o.client.GetObjectMetadata(input)
	if err != nil {
		if oriErr, ok := errors.Cause(err).(obs.ObsError); ok {
			log.WithFields(
				logrus.Fields{
					"code":    oriErr.Code,
					"message": oriErr.Message,
				},
			).Debugf("obs err.Error contents (origErr=%v)", oriErr.Error())

			if oriErr.Status == "404 Not Found" {
				log.Debug("Object doesn't exist - got not found")
				return false, nil
			}
		}
		return false, errors.Wrap(err, "failed to get object metadata")
	}

	log.Debug("Object exists")
	return true, nil
}

func (o *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	input := &obs.GetObjectInput{}
	input.Bucket = bucket
	input.Key = key
	output, err := o.client.GetObject(input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get object %s", key)
	}
	return output.Body, nil
}

func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	var ret []string
	var marker string
	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.Prefix = prefix
	input.Delimiter = delimiter
	for {
		input.Marker = marker
		output, err := o.client.ListObjects(input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list object")
		}
		ret = append(ret, output.CommonPrefixes...)
		if output.IsTruncated {
			marker = output.NextMarker
		} else {
			break
		}
	}
	return ret, nil
}

func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	var ret []string
	var marker string
	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.Prefix = prefix
	for {
		input.Marker = marker
		output, err := o.client.ListObjects(input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list object")
		}
		for _, content := range output.Contents {
			ret = append(ret, content.Key)
		}
		if output.IsTruncated {
			marker = output.NextMarker
		} else {
			break
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ret)))
	return ret, nil
}

func (o *ObjectStore) DeleteObject(bucket, key string) error {
	input := &obs.DeleteObjectInput{}
	input.Bucket = bucket
	input.Key = key
	_, err := o.client.DeleteObject(input)
	return errors.Wrapf(err, "failed to delete object %s", key)
}

func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	input := &obs.CreateSignedUrlInput{}
	input.Bucket = bucket
	input.Key = key
	input.Expires = int(ttl.Seconds())
	input.Method = obs.HttpMethodGet
	output, err := o.client.CreateSignedUrl(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to create signedURL")
	}
	return output.SignedUrl, nil
}
