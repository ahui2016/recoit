package ibm

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/IBM/ibm-cos-sdk-go/aws"
	"github.com/IBM/ibm-cos-sdk-go/aws/credentials/ibmiam"
	"github.com/IBM/ibm-cos-sdk-go/aws/session"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/ahui2016/recoit/util"
)

const authEndpoint = "https://iam.cloud.ibm.com/identity/token"

// COS 包含 IBM COS 的相关信息，便于相关操作。
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

// NewCOS 生成一个 cos, 在 package main 里作为一个全局变量来使用。
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
		// WithRegion(cos.bucketLocation).
		WithEndpoint(cos.serviceEndpoint).
		WithCredentials(ibmiam.NewStaticCredentials(
			aws.NewConfig(), cos.authEndpoint, cos.apiKey, cos.serviceInstanceID)).
		WithS3ForcePathStyle(true)

	sess := session.Must(session.NewSession())
	cos.client = s3.New(sess, cos.conf)
}

func (cos *COS) makeSureConfig() {
	if cos.conf == nil {
		cos.makeConfig()
	}
}

/*
func (cos *COS) Upload(objName string, objBody io.Reader) (*s3manager.UploadOutput, error) {
	cos.makeSureConfig()
	return cos.upload(objName, objBody)
}

func (cos *COS) UploadFile(localFile string) (*s3manager.UploadOutput, error) {
	cos.makeSureConfig()
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
*/

// PutObject 上传数据到云端。
func (cos *COS) PutObject(objName string, objBody io.ReadSeeker) (*s3.PutObjectOutput, error) {
	cos.makeSureConfig()
	return cos.putObject(objName, objBody)
}

// PutFile 上传本地文件到云端。
func (cos *COS) PutFile(localFile string) (*s3.PutObjectOutput, error) {
	cos.makeSureConfig()
	return cos.putFile(localFile)
}

func (cos *COS) putFile(localFile string) (*s3.PutObjectOutput, error) {
	file, err := os.Open(localFile)
	if err != nil {
		return nil, err
	}
	//noinspection GoUnhandledErrorResult
	defer file.Close()
	name := filepath.Base(localFile)
	return cos.putObject(name, file)
}

func (cos *COS) putObject(objName string, objBody io.ReadSeeker) (*s3.PutObjectOutput, error) {
	input := s3.PutObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(objName),
		Body:   objBody,
	}
	return cos.client.PutObject(&input)
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
	cos.makeSureConfig()
	output, err := cos.getObject(name)
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

// DeleteObject 删除云端的一个对象。
func (cos *COS) DeleteObject(name string) (*s3.DeleteObjectOutput, error) {
	cos.makeSureConfig()
	return cos.deleteObject(name)
}

func (cos *COS) deleteObject(name string) (*s3.DeleteObjectOutput, error) {
	input := s3.DeleteObjectInput{
		Bucket: aws.String(cos.bucketName),
		Key:    aws.String(name),
	}
	return cos.client.DeleteObject(&input)
}

// TryUploadDelete 尝试上传一个对象到云端，然后下载同一个对象，并对比两者是否相同。
func (cos *COS) TryUploadDelete() error {
	cos.makeSureConfig()
	key := "TempObject.try"
	randomContent := util.NewID() + util.TimeNow()
	content := strings.NewReader(randomContent)

	// 上传
	if _, err := cos.putObject(key, content); err != nil {
		return err
	}

	// 下载
	output, err := cos.getObject(key)
	// 下载后立即删除，不管下载是否成功都删除。
	if _, errDel := cos.deleteObject(key); errDel != nil {
		return errDel
	}
	if err != nil {
		return err
	}
	defer output.Body.Close()

	// 对比
	body, err := ioutil.ReadAll(output.Body)
	if string(body) != randomContent {
		return errors.New("the downloaded object is not equal to the uploaded object")
	}
	return nil
}
