package mssh

import (
	"fmt"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
)

type Cli struct {
	User       string
	Pwd        string
	keyPath    string
	Addr       string
	Signer     string
	client     *gossh.Client
	session    *gossh.Session
	LastResult string
}

func (c *Cli) GetAuth() ([]gossh.AuthMethod, error) {
	if c.keyPath != "" {
		key, err := ioutil.ReadFile(c.keyPath)
		if err != nil {
			return nil, err
		}
		signer, err := gossh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	} else if c.Pwd != "" {
		return []gossh.AuthMethod{gossh.Password(c.Pwd)}, nil
	} else if c.Signer != "" {
		signer, err := gossh.ParsePrivateKey([]byte(c.Signer))
		if err != nil {
			return nil, err
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	} else {
		return []gossh.AuthMethod{}, nil
	}
}

func (c *Cli) Connect() (*Cli, error) {
	config := &gossh.ClientConfig{}
	config.SetDefaults()
	config.User = c.User
	auth, err := c.GetAuth()
	if err != nil {
		return nil, err
	}
	config.Auth = auth
	config.HostKeyCallback = func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		return nil
	}
	client, err := gossh.Dial("tcp", c.Addr, config)
	if err != nil {
		return c, err
	}
	c.client = client
	return c, err
}

func (c Cli) Run(command string) (string, error) {
	if c.client == nil {
		if _, err := c.Connect(); err != nil {
			return "", err
		}
	}
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	buf, err := session.CombinedOutput(command)
	c.LastResult = string(buf)
	return c.LastResult, err
}

func (c Cli) UploadFile(localFilePath string, remotePath string) error {
	ftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer ftpClient.Close()
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		fmt.Println("open:", err)
		return err
	}
	defer srcFile.Close()
	dstFile, err := ftpClient.Create(path.Join(remotePath, path.Base(localFilePath)))
	if err != nil {
		fmt.Println("create", err)
		return err
	}
	defer dstFile.Close()
	size, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		fmt.Println("readfrom", err)
		return err
	}
	fmt.Println(size)
	return nil
}

func (c Cli) UploadFileAndProgress(srcFile io.Reader, remoteFile string, ch chan<- int64) error {
	ftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer ftpClient.Close()
	dstFile, err := ftpClient.Create(remoteFile)
	if err != nil {
		fmt.Println("create", err)
		return err
	}
	defer dstFile.Close()
	buf := make([]byte, 1<<15) // 每个数据包最大支持32kb
	for {
		n, err := srcFile.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
				return err
			} else {
				break
			}
		}
		size, _ := dstFile.Write(buf[:n])
		//fmt.Println(size)
		//ds, _ := dstFile.Stat()
		ch <- int64(size)
	}
	return nil
}

func (c *Cli) DownloadFile(remotePath string, localDir string, localFileName string) error {
	if localFileName == "" {
		localFileName = path.Base(remotePath)
	}
	ftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer ftpClient.Close()
	srcFile, err := ftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(path.Join(localDir, localFileName))
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if _, err := srcFile.WriteTo(dstFile); err != nil {
		return err
	}
	return nil
}
