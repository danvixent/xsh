package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/danvixent/sshx/util"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

type Plan struct {
	PlainHosts           []string
	Command              string
	SSHKeyPath           string
	Output               io.WriteCloser
	CertificateAlgorithm string
	ParallelLimit        *int

	hosts    []Host
	errgroup errgroup.Group
	stop     chan struct{}
}

type SSHClient struct {
	conn    ssh.Conn
	channel ssh.NewChannel
	request ssh.Request
}

type Host struct {
	user string
	host string

	session *ssh.Session
}

var (
	dialTimeout = time.Second * 10

	ErrNoHosts        = errors.New("no hosts specified")
	ErrNoSSHKeysFound = errors.New("no ssh keys found in default directory")

	beginBytes = []byte(`-----BEGIN`)

	ignoreFiles = map[string]struct{}{
		"config":          {},
		"known_hosts.old": {},
		"known_hosts":     {},
	}

	publicKeyRegex = regexp.MustCompile(`\.pub$`)

	timeout = time.Second * 10
)

const (
	defaultSSHConfigDir = "~/.ssh/"
)

func RunCommand() {
}

func (p *Plan) OpenConns() error {
	if len(p.PlainHosts) == 0 {
		return ErrNoHosts
	}

	for _, host := range p.PlainHosts {
		parts := strings.Split(host, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid host: %s, hosts must be in the format user@host", host)
		}

		signers, err := p.getSigners(p.SSHKeyPath)
		if err != nil {
			return fmt.Errorf("failed to get signers: %v", err)
		}

		h := Host{
			user: parts[0],
			host: parts[1],
		}

		cfg := &ssh.ClientConfig{
			Config:         ssh.Config{},
			User:           h.user,
			Auth:           []ssh.AuthMethod{ssh.PublicKeys(signers...)},
			BannerCallback: ssh.BannerDisplayStderr(),
			Timeout:        timeout,
		}

		sshConn, err := ssh.Dial("tcp", h.host, cfg)
		if err != nil {
			return fmt.Errorf("failed to dial SSH for host %s: %v", host, err)
		}

		session, err := sshConn.NewSession()
		if err != nil {
			return fmt.Errorf("failed to start ssh session for host %s: %v", host, err)
		}

		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		// Request pseudo terminal
		if err := h.session.RequestPty("xterm", 40, 80, modes); err != nil {
			return fmt.Errorf("failed to set request terminal for host %s: %v", h.host, err)
		}
		// Start remote shell
		if err := h.session.Shell(); err != nil {
			return fmt.Errorf("failed to start shell for host %s: %v", h.host, err)
		}

		h.session = session
		p.hosts = append(p.hosts)
	}

	go p.listenForClose()

	return nil
}

func (p *Plan) Execute(ctx context.Context) (*Result, error) {
	result := &Result{}

	if p.ParallelLimit != nil {
		err := p.executeErrG(ctx, result)
		return result, err
	}

	err := p.executeWG(ctx, result)
	return result, err
}

// executes with a waitgroup
func (p *Plan) executeWG(ctx context.Context, result *Result) error {
	var wg sync.WaitGroup

	for _, h := range p.hosts {
		wg.Add(1)
		go func(session *ssh.Session, host string, result *Result) {
			defer wg.Done()

			start := time.Now()
			out, err := session.Output(p.Command)
			result.AddResult(start, time.Now(), host, out, err)
		}(h.session, h.host, result)
	}

	wg.Wait()
	return nil
}

// executes with a errgroup  limiting concurrency
func (p *Plan) executeErrG(ctx context.Context, result *Result) error {
	errg := &errgroup.Group{}
	errg.SetLimit(*p.ParallelLimit)

	for _, h := range p.hosts {
		session := h.session
		host := h.host
		errg.Go(func() error {
			start := time.Now()
			out, err := session.Output(p.Command)
			result.AddResult(start, time.Now(), host, out, err)
			return nil
		})

	}

	return nil
}

func (p *Plan) Close(ctx context.Context) {
	select {
	case <-ctx.Done():
		close(p.stop)
	}
}

func (p *Plan) listenForClose() {
	<-p.stop
	for i := range p.hosts {
		_ = p.hosts[i].session.Close()
	}
}

func (p *Plan) getSigners(keyFile string) ([]ssh.Signer, error) {
	if !util.IsStringEmpty(keyFile) {
		f, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(f)
		if err != nil {
			log.Fatalf("parse ssh private key failed:%v", err)
		}

		return []ssh.Signer{signer}, nil
	}

	var signers []ssh.Signer
	err := filepath.Walk(defaultSSHConfigDir, func(path string, info fs.FileInfo, err error) error {
		_, ok := ignoreFiles[info.Name()]
		if ok || publicKeyRegex.MatchString(info.Name()) {
			// skip config files
			return nil
		}

		f, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read key file: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(f)
		if err != nil {
			log.Fatalf("parse key failed:%v", err)
		}

		signers = append(signers, signer)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(signers) == 0 {
		return nil, ErrNoSSHKeysFound
	}

	return signers, nil
}
