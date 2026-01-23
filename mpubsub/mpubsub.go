package mpubsub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type SubGroupIface[T any] interface {
	ID() string
	Write(T) error
}

type Message[T any] struct {
	GroupId string
	Data    T
}

// 消息队列代理

type MPubSub[T any] struct {
	channel   []string
	chanNext  chan string
	subFunc   func(ctx context.Context, k string) (<-chan []byte, error)
	pubFunc   func(ctx context.Context, k string, m []byte) error //
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	subGroup  sync.Map // channelId ==>  SubGroupIface
	parser    *GobParser
	isRunning atomic.Bool
	logger    LoggerIface
}

func NewMPubSub[T any](channel []string, subFunc func(ctx context.Context, k string) (<-chan []byte, error),
	pubFunc func(ctx context.Context, k string, m []byte) error, opts ...Option[T]) (*MPubSub[T], error) {
	if len(channel) == 0 {
		return nil, fmt.Errorf("channel err")
	}
	ctx, chancel := context.WithCancel(context.Background())
	m := &MPubSub[T]{
		channel:  channel,
		chanNext: make(chan string),
		subFunc:  subFunc,
		pubFunc:  pubFunc,
		ctx:      ctx,
		cancel:   chancel,
		parser:   NewGobPaser(),
		logger:   _log,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

type Option[T any] func(sub *MPubSub[T])

func WithLogger[T any](logger LoggerIface) Option[T] {
	return func(m *MPubSub[T]) {
		m.logger = logger
	}
}

// Register 注册订阅组 到代理
func (m *MPubSub[T]) Register(channel SubGroupIface[T]) {
	if _, e := m.subGroup.Load(channel.ID()); !e {
		m.subGroup.Store(channel.ID(), channel)
	}
}

func (m *MPubSub[T]) UnRegister(channelId string) {
	m.subGroup.Delete(channelId)
}

func (m *MPubSub[T]) Start() {
	if !m.isRunning.CompareAndSwap(false, true) {
		return
	}
	for _, k := range m.channel {
		m.startChannel(k)
	}
	// 频道策略
	go m.chanelStart()
	return
}

func (m *MPubSub[T]) startChannel(k string) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		retryCount := 0
		for m.isRunning.Load() {
			select {
			case <-m.ctx.Done():
				return
			default:
			}
			if retryCount > 0 {
				time.Sleep(3 * time.Second)
			}
			if err := m.listenChannel(k); err != nil {
				if retryCount < 10 {
					retryCount++
					continue
				} else {
					m.logger.Error(fmt.Errorf("监听消息超过最大重试次数 c频道:%s", k))
					return
				}
			}
			retryCount = 0
		}

	}()
}

func (m *MPubSub[T]) listenChannel(channel string) error {
	msgChan, err := m.subFunc(m.ctx, channel)
	if err != nil {
		return err
	}
	for {
		select {
		case <-m.ctx.Done():
			return nil
		case msgBytes, ok := <-msgChan:
			if !ok {
				return fmt.Errorf("channel Closed %s", channel)
			}
			m.handleMessage(msgBytes)
		}
	}
}

func (m *MPubSub[T]) handleMessage(msgBytes []byte) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error(fmt.Errorf("panic in handle message %v", r))
		}
	}()
	var msg Message[T]
	if err := m.parser.Decoder(msgBytes, &msg); err != nil {
		m.logger.Error(fmt.Errorf("failed to decode message err: %v", err))
		return
	}
	if val, ok := m.subGroup.Load(msg.GroupId); ok {
		subscribers := val.(SubGroupIface[T])
		if err := subscribers.Write(msg.Data); err != nil {
			m.logger.Error(fmt.Errorf("failed to write to subGroup %v", err))
		}
	}
}

func (m *MPubSub[T]) Publish(msg Message[T]) error {
	if !m.isRunning.Load() {
		return fmt.Errorf("ErrNotRunning")
	}
	data, err := m.parser.Encoder(msg)
	if err != nil {
		return err
	}
	channel := m.channelNext()
	if err := m.pubFunc(m.ctx, channel, data); err != nil {
		return err
	}
	return nil
}

// 消息发送
func (m *MPubSub[T]) chanelStart() {
	for {
		for _, v := range m.channel {
			select {
			case m.chanNext <- v:
			case <-m.ctx.Done():
				return
			}
		}
	}
}

func (m *MPubSub[T]) channelNext() string {
	return <-m.chanNext
}

func (m *MPubSub[T]) publishWithRetry(ctx context.Context, channel string, data []byte) error {
	var lastErr error
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
		}
		if i > 0 {
			time.Sleep(3 * time.Second)
		}
		if err := m.pubFunc(ctx, channel, data); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return lastErr
}

// 判断监听组是否存在
func (m *MPubSub[T]) ExistsSubGroup(groupId string) bool {
	if _, exists := m.subGroup.Load(groupId); exists {
		return true
	}
	return false
}

func (m *MPubSub[T]) GetSubGroup(groupId string) (SubGroupIface[T], bool) {
	if v, exists := m.subGroup.Load(groupId); exists {
		return v.(SubGroupIface[T]), true
	}
	return nil, false
}
