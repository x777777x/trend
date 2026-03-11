package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// Client 包装 etcd 客户端及通用操作
type Client struct {
	Cli *clientv3.Client
}

// Config 定义 etcd 配置
type Config struct {
	Endpoints   []string `yaml:"endpoints"`
	DialTimeout int      `yaml:"dial_timeout"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
}

// NewClient 创建并返回一个新的 etcd Client 实例
func NewClient(cfg *Config) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: time.Duration(cfg.DialTimeout) * time.Second,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, err
	}
	return &Client{Cli: cli}, nil
}

// Close 关闭 etcd 客户端
func (c *Client) Close() error {
	if c.Cli != nil {
		return c.Cli.Close()
	}
	return nil
}

// NewSession 创建用于分布式锁或选举的并发会话 (Session)
func (c *Client) NewSession() (*concurrency.Session, error) {
	return concurrency.NewSession(c.Cli, concurrency.WithTTL(5))
}

// Put 封装 Put 操作
func (c *Client) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	return c.Cli.Put(ctx, key, val, opts...)
}

// Get 封装 Get 操作
func (c *Client) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	return c.Cli.Get(ctx, key, opts...)
}

// Watch 封装 Watch 操作
func (c *Client) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return c.Cli.Watch(ctx, key, opts...)
}
