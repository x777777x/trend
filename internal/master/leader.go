package master

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.etcd.io/etcd/client/v3/concurrency"

	"trend/pkg/etcd"
	"trend/pkg/logger"
)

// LeaderElection 管理 Master 节点的选举状态
type LeaderElection struct {
	client     *etcd.Client
	session    *concurrency.Session
	election   *concurrency.Election
	prefix     string
	id         string
	isLeader   atomic.Bool
	cancelFunc context.CancelFunc // 用于取消选举相关的后台操作
	done       chan struct{}      // 信号后台 goroutine 已退出
}

// NewLeaderElection 初始化一个 LeaderElection 对象
func NewLeaderElection(client *etcd.Client, id string, cluster string) *LeaderElection {
	return &LeaderElection{
		client: client,
		prefix: fmt.Sprintf("/trend/master/%s/leader", cluster),
		id:     id,
	}
}

// Start 开始参与选举
func (le *LeaderElection) Start(ctx context.Context) error {
	sess, err := le.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	le.session = sess
	le.election = concurrency.NewElection(sess, le.prefix)

	// 后台开启一个 goroutine 竞争 leader
	electionCtx, cancel := context.WithCancel(ctx)
	le.cancelFunc = cancel
	le.done = make(chan struct{})

	go func() {
		defer close(le.done)
		for {
			select {
			case <-electionCtx.Done():
				logger.Info("Stop participating in election")
				return
			default:
				logger.Info("Attempting to campaign for leader...")
				// Campaign 阻塞直到成为 Leader 或发生错误
				if err := le.election.Campaign(electionCtx, le.id); err != nil {
					logger.Error("Campaign failed, retrying...", logger.Err(err))
					time.Sleep(2 * time.Second)
					continue
				}

				// 成功当选
				le.isLeader.Store(true)
				logger.Info("Successfully elected as Leader!")

				// 阻塞，直到上下文被取消，或者 session 丢失 (比如 etcd 连接断开)
				select {
				case <-electionCtx.Done():
					return
				case <-sess.Done():
					logger.Warn("Session lost. Lost leader status.")
					le.isLeader.Store(false)
				}
			}
		}
	}()

	return nil
}

// IsLeader 返回当前节点是否是 Leader
func (le *LeaderElection) IsLeader() bool {
	return le.isLeader.Load()
}

// Stop 停止参与选举，并如果当前是 Leader 则进行释放
func (le *LeaderElection) Stop() {
	if le.cancelFunc != nil {
		le.cancelFunc()
	}
	// 等待后台 goroutine 退出，避免 resign 竞态
	if le.done != nil {
		<-le.done
	}
	le.resign()
	if le.session != nil {
		le.session.Close()
	}
}

// resign 内部方法，辞去 leader 职位
func (le *LeaderElection) resign() {
	if le.isLeader.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := le.election.Resign(ctx); err != nil {
			logger.Error("Failed to resign from leader", logger.Err(err))
		} else {
			le.isLeader.Store(false)
			logger.Info("Resigned from leader successfully")
		}
	}
}
