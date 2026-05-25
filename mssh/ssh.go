package mssh

import (
	"errors"
	"io"
	"net"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

// Config SSH 连接配置
type Config struct {
	// 用户名
	User string
	// 密码（与 KeyPath/KeyPEM 三选一）
	Password string
	// 私钥文件路径
	KeyPath string
	// 私钥 PEM 内容（字符串形式，与 KeyPath 二选一）
	KeyPEM string
	// 服务器地址，格式 host:port
	Addr string
	// 连接超时，默认 10s
	Timeout time.Duration
}

// Client SSH 客户端
type Client struct {
	config *Config
	conn   *gossh.Client
}

// New 创建 SSH 客户端并建立连接。
func New(cfg *Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	auth, err := buildAuth(cfg)
	if err != nil {
		return nil, err
	}

	sshCfg := &gossh.ClientConfig{
		User: cfg.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key gossh.PublicKey) error {
			return nil
		},
		Timeout: cfg.Timeout,
	}

	conn, err := gossh.Dial("tcp", cfg.Addr, sshCfg)
	if err != nil {
		return nil, err
	}
	return &Client{config: cfg, conn: conn}, nil
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Run 执行非交互命令，返回合并后的 stdout+stderr 输出。
// 不支持 vim/top 等需要 PTY 的程序，请使用 Shell。
func (c *Client) Run(command string) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(command)
	return string(out), err
}

// PTYOptions PTY 终端参数
type PTYOptions struct {
	// 终端类型，默认 xterm-256color
	Term string
	// 宽度（列），默认 220
	Width int
	// 高度（行），默认 50
	Height int
}

func (o *PTYOptions) fill() {
	if o.Term == "" {
		o.Term = "xterm-256color"
	}
	if o.Width <= 0 {
		o.Width = 220
	}
	if o.Height <= 0 {
		o.Height = 50
	}
}

// ShellOptions Shell 会话选项
type ShellOptions struct {
	PTY PTYOptions
	// 标准输入，默认 os.Stdin
	Stdin io.Reader
	// 标准输出，默认 os.Stdout
	Stdout io.Writer
	// 标准错误，默认 os.Stderr
	Stderr io.Writer
}

// Shell 开启交互式 Shell，支持 vim/top 等需要 PTY 的程序。
// 调用方负责将本地终端切换为 raw 模式（见示例）。
// 函数阻塞直到 Shell 退出。
func (c *Client) Shell(opts *ShellOptions) error {
	if opts == nil {
		opts = &ShellOptions{}
	}
	opts.PTY.fill()

	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr

	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	if err := sess.RequestPty(opts.PTY.Term, opts.PTY.Height, opts.PTY.Width, modes); err != nil {
		return err
	}

	if err := sess.Shell(); err != nil {
		return err
	}

	return sess.Wait()
}

// ResizeOption 用于 WindowChange
type ResizeOption struct {
	Width  int
	Height int
}

// Session 可控制的交互会话（支持动态调整窗口大小）
type Session struct {
	sess *gossh.Session
}

// NewSession 创建可手动控制的交互会话。
// 适合需要动态 resize（如 WebSocket 终端代理）的场景。
func (c *Client) NewSession(opts *ShellOptions) (*Session, error) {
	if opts == nil {
		opts = &ShellOptions{}
	}
	opts.PTY.fill()

	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	sess, err := c.conn.NewSession()
	if err != nil {
		return nil, err
	}

	sess.Stdin = stdin
	sess.Stdout = stdout
	sess.Stderr = stderr

	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	if err := sess.RequestPty(opts.PTY.Term, opts.PTY.Height, opts.PTY.Width, modes); err != nil {
		sess.Close()
		return nil, err
	}

	if err := sess.Shell(); err != nil {
		sess.Close()
		return nil, err
	}

	return &Session{sess: sess}, nil
}

// Resize 动态调整终端窗口大小（用于 WebSocket 终端 resize 事件）。
func (s *Session) Resize(width, height int) error {
	return s.sess.WindowChange(height, width)
}

// Wait 等待会话结束。
func (s *Session) Wait() error {
	return s.sess.Wait()
}

// Close 关闭会话。
func (s *Session) Close() error {
	return s.sess.Close()
}

// ========== 文件传输 ==========

// UploadOption 上传选项
type UploadOption struct {
	// 上传进度回调，参数为本次写入字节数；为 nil 时不回调
	OnProgress func(written int64)
	// 进度回调间隔，默认 500ms
	ProgressInterval time.Duration
}

// Upload 上传本地文件到远程路径。
// remotePath 可以是目录（自动取本地文件名）或完整文件路径。
func (c *Client) Upload(localPath, remotePath string, opt *UploadOption) error {
	fc, err := sftp.NewClient(c.conn,
		sftp.UseConcurrentWrites(true),
		sftp.MaxPacketUnchecked(1<<16),
	)
	if err != nil {
		return err
	}
	defer fc.Close()

	// 若 remotePath 是目录则拼接文件名
	info, err := fc.Stat(remotePath)
	if err == nil && info.IsDir() {
		remotePath = path.Join(remotePath, path.Base(localPath))
	}

	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := fc.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if opt == nil || opt.OnProgress == nil {
		_, err = dst.ReadFrom(src)
		return err
	}

	// 带进度回调
	interval := opt.ProgressInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	errCh := make(chan error, 1)
	go func() {
		_, e := dst.ReadFrom(src)
		errCh <- e
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	last := int64(0)

	for {
		select {
		case <-ticker.C:
			fi, e := dst.Stat()
			if e != nil {
				return e
			}
			diff := fi.Size() - last
			last = fi.Size()
			if diff > 0 {
				opt.OnProgress(diff)
			}
		case e := <-errCh:
			return e
		}
	}
}

// UploadReader 从 io.Reader 上传到远程完整文件路径。
func (c *Client) UploadReader(src io.Reader, remotePath string, opt *UploadOption) error {
	fc, err := sftp.NewClient(c.conn,
		sftp.UseConcurrentWrites(true),
		sftp.MaxPacketUnchecked(1<<16),
	)
	if err != nil {
		return err
	}
	defer fc.Close()

	dst, err := fc.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if opt == nil || opt.OnProgress == nil {
		_, err = io.Copy(dst, src)
		return err
	}

	interval := opt.ProgressInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	errCh := make(chan error, 1)
	go func() {
		_, e := io.Copy(dst, src)
		errCh <- e
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	last := int64(0)

	for {
		select {
		case <-ticker.C:
			fi, e := dst.Stat()
			if e != nil {
				return e
			}
			diff := fi.Size() - last
			last = fi.Size()
			if diff > 0 {
				opt.OnProgress(diff)
			}
		case e := <-errCh:
			return e
		}
	}
}

// Download 下载远程文件到本地路径。
// localPath 可以是目录（自动取远程文件名）或完整文件路径。
func (c *Client) Download(remotePath, localPath string, opt *UploadOption) error {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer fc.Close()

	// 若 localPath 是目录则拼接文件名
	if fi, e := os.Stat(localPath); e == nil && fi.IsDir() {
		localPath = path.Join(localPath, path.Base(remotePath))
	}

	src, err := fc.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if opt == nil || opt.OnProgress == nil {
		_, err = io.Copy(dst, src)
		return err
	}

	interval := opt.ProgressInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}

	pw := &progressWriter{w: dst, fn: opt.OnProgress}
	_, err = io.Copy(pw, src)
	return err
}

// DownloadWriter 下载远程文件写入 io.Writer。
func (c *Client) DownloadWriter(remotePath string, dst io.Writer, opt *UploadOption) error {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer fc.Close()

	src, err := fc.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	if opt == nil || opt.OnProgress == nil {
		_, err = io.Copy(dst, src)
		return err
	}

	pw := &progressWriter{w: dst, fn: opt.OnProgress}
	_, err = io.Copy(pw, src)
	return err
}

// Mkdir 在远程创建目录（含父目录）。
func (c *Client) Mkdir(remotePath string) error {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer fc.Close()
	return fc.MkdirAll(remotePath)
}

// Remove 删除远程文件。
func (c *Client) Remove(remotePath string) error {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return err
	}
	defer fc.Close()
	return fc.Remove(remotePath)
}

// Stat 获取远程文件信息。
func (c *Client) Stat(remotePath string) (os.FileInfo, error) {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return nil, err
	}
	defer fc.Close()
	return fc.Stat(remotePath)
}

// ReadDir 列出远程目录内容。
func (c *Client) ReadDir(remotePath string) ([]os.FileInfo, error) {
	fc, err := sftp.NewClient(c.conn)
	if err != nil {
		return nil, err
	}
	defer fc.Close()
	entries, err := fc.ReadDir(remotePath)
	if err != nil {
		return nil, err
	}
	out := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		out[i] = e
	}
	return out, nil
}

// ========== 内部工具 ==========

func buildAuth(cfg *Config) ([]gossh.AuthMethod, error) {
	if cfg.KeyPath != "" {
		key, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		signer, err := gossh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	}
	if cfg.KeyPEM != "" {
		signer, err := gossh.ParsePrivateKey([]byte(cfg.KeyPEM))
		if err != nil {
			return nil, err
		}
		return []gossh.AuthMethod{gossh.PublicKeys(signer)}, nil
	}
	if cfg.Password != "" {
		return []gossh.AuthMethod{gossh.Password(cfg.Password)}, nil
	}
	return nil, errors.New("mssh: no auth method provided")
}

type progressWriter struct {
	w  io.Writer
	fn func(int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if n > 0 {
		pw.fn(int64(n))
	}
	return n, err
}
