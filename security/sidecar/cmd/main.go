package main

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/security/manager/apis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"os"
	"path"
	"time"
)

type SidecarConfig struct {
	CertDir string
}

const (
	certFile = "/etc/certs/cert.pem"
	keyFile  = "/etc/certs/key.pem"
	caFile   = "/etc/certs/ca.pem"
)

func main() {
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		logrus.Fatalf("please provide %s", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		logrus.Fatalf("please provide %s", keyFile)
	}
	if _, err := os.Stat(caFile); os.IsNotExist(err) {
		logrus.Fatalf("please provide %s", caFile)
	}

	cfg := SidecarConfig{
		CertDir: "/tmp/certs",
	}

	if err := copy(certFile, path.Join(cfg.CertDir, "cert.pem")); err != nil {
		logrus.Fatalf(err.Error())
	}

	if err := copy(keyFile, path.Join(cfg.CertDir, "key.pem")); err != nil {
		logrus.Fatalf(err.Error())
	}

	if err := copy(caFile, path.Join(cfg.CertDir, "ca.pem")); err != nil {
		logrus.Fatalf(err.Error())
	}

	for {
		conn, err := grpc.Dial("localhost:3232")
		if err != nil {
			logrus.Error(err)
			<-time.After(300 * time.Millisecond)
			continue
		}

		mClient := manager.NewManagerClient(conn)
		mClient.CertificatesUpdated(context.Background(), &empty.Empty{})

		conn.Close()
		break
	}

}

func copy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}
