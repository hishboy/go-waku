package rpc

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_filter/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"go.uber.org/zap"
)

type FilterService struct {
	node *node.WakuNode
	log  *zap.Logger

	messages      map[string][]*wpb.WakuMessage
	cacheCapacity int
	messagesMutex sync.RWMutex

	runner *runnerService
}

type FilterContentArgs struct {
	Topic          string                            `json:"topic,omitempty"`
	ContentFilters []*pb.FilterRequest_ContentFilter `json:"contentFilters,omitempty"`
}

type ContentTopicArgs struct {
	ContentTopic string `json:"contentTopic,omitempty"`
}

func NewFilterService(node *node.WakuNode, cacheCapacity int, log *zap.Logger) *FilterService {
	s := &FilterService{
		node:          node,
		log:           log.Named("filter"),
		cacheCapacity: cacheCapacity,
		messages:      make(map[string][]*wpb.WakuMessage),
	}
	s.runner = newRunnerService(node.Broadcaster(), s.addEnvelope)
	return s
}

func makeContentFilter(args *FilterContentArgs) legacy_filter.ContentFilter {
	var contentTopics []string
	for _, contentFilter := range args.ContentFilters {
		contentTopics = append(contentTopics, contentFilter.ContentTopic)
	}

	return legacy_filter.ContentFilter{
		Topic:         args.Topic,
		ContentTopics: contentTopics,
	}
}

func (f *FilterService) addEnvelope(envelope *protocol.Envelope) {
	f.messagesMutex.Lock()
	defer f.messagesMutex.Unlock()

	contentTopic := envelope.Message().ContentTopic
	if _, ok := f.messages[contentTopic]; !ok {
		return
	}

	// Keep a specific max number of messages per topic
	if len(f.messages[envelope.PubsubTopic()]) >= f.cacheCapacity {
		f.messages[envelope.PubsubTopic()] = f.messages[envelope.PubsubTopic()][1:]
	}

	f.messages[contentTopic] = append(f.messages[contentTopic], envelope.Message())
}

func (f *FilterService) Start() {
	f.runner.Start()
}

func (f *FilterService) Stop() {
	f.runner.Stop()
}

func (f *FilterService) PostV1Subscription(req *http.Request, args *FilterContentArgs, reply *SuccessReply) error {
	_, _, err := f.node.LegacyFilter().Subscribe(
		req.Context(),
		makeContentFilter(args),
		legacy_filter.WithAutomaticPeerSelection(),
	)
	if err != nil {
		f.log.Error("subscribing to topic", zap.String("topic", args.Topic), zap.Error(err))
		return err
	}
	for _, contentFilter := range args.ContentFilters {
		f.messages[contentFilter.ContentTopic] = make([]*wpb.WakuMessage, 0)
	}

	*reply = true
	return nil
}

func (f *FilterService) DeleteV1Subscription(req *http.Request, args *FilterContentArgs, reply *SuccessReply) error {
	err := f.node.LegacyFilter().UnsubscribeFilter(
		req.Context(),
		makeContentFilter(args),
	)
	if err != nil {
		f.log.Error("unsubscribing from topic", zap.String("topic", args.Topic), zap.Error(err))
		return err
	}
	for _, contentFilter := range args.ContentFilters {
		delete(f.messages, contentFilter.ContentTopic)
	}

	*reply = true
	return nil
}

func (f *FilterService) GetV1Messages(req *http.Request, args *ContentTopicArgs, reply *MessagesReply) error {
	f.messagesMutex.Lock()
	defer f.messagesMutex.Unlock()

	if _, ok := f.messages[args.ContentTopic]; !ok {
		return fmt.Errorf("topic %s not subscribed", args.ContentTopic)
	}

	for i := range f.messages[args.ContentTopic] {
		msg := f.messages[args.ContentTopic][i]
		rpcMsg, err := ProtoToRPC(msg)
		if err != nil {
			f.log.Warn("could not include message in response", zap.Error(err))
		} else {
			*reply = append(*reply, rpcMsg)
		}
	}

	f.messages[args.ContentTopic] = make([]*wpb.WakuMessage, 0)
	return nil
}
