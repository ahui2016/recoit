package ibm

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/IBM/ibm-cos-sdk-go/service/s3/s3manager"
)

const authEndpoint = "https://iam.cloud.ibm.com/identity/token"

type COS struct {
	apiKey            string
	serviceInstanceID string // resource_instance_id
	authEndpoint      string
	serviceEndpoint   string
	bucketLocation    string
	bucketName        string
	conf              *aws.Config
	client            *s3.S3
}

func NewCOS(settings *Settings) *COS {
	return &COS{
		apiKey:            settings.ApiKey,
		serviceInstanceID: settings.ServiceInstanceID,
		authEndpoint:      authEndpoint, // const
		serviceEndpoint:   settings.ServiceEndpoint,
		bucketLocation:    settings.BucketLocation,
		bucketName:        settings.BucketName,
	}
}

func (cos *COS) makeConfig() {
	log.Println("making IBM COS config...")
	cos.conf = aws.NewConfig().
		WithEndpoint(cos.serviceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(
			aws.NewConfig(), cos.authEndpoint, cos.apiKey, cos.serviceInstanceID)).
		WithS3ForcePathStyle(true)

	sess := session.Must(session.NewSession())
	cos.client = s3.New(sess, cos.conf)
}

func (cos *COS) Upload(objName string, objBody io.Reader) (*s3manager.UploadOutput, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	return cos.upload(objName, objBody)
}

func (cos *COS) UploadFile(localFile string) (*s3manager.UploadOutput, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	return cos.uploadFile(localFile)
}

func (cos *COS) uploadFile(localFile string) (*s3manager.UploadOutput, error) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()
	name := filepath.Base(localFile)
	return cos.upload(name, file)
}

func (cos *COS) upload(objName string, objBody io.Reader) (*s3manager.UploadOutput, error) {
	sess := session.Must(session.NewSession(cos.conf))
	uploader := s3manager.NewUploader(sess)
	return uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(objName),
		Body:   objBody,
	})
}

func (cos *COS) getObject(name string) (*s3.GetObjectOutput, error) {
	input := s3.GetObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(name),
	}
	return cos.client.GetObject(&input)
}

// GetObjectBody 返回 io.ReadCloser, 要记得关闭资源.
func (cos *COS) GetObjectBody(name string) (io.ReadCloser, error) {
	if cos.conf == nil {
		cos.makeConfig()
	}
	output, err := cos.getObject(name)
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}
