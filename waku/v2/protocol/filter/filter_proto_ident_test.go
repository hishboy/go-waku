package filter

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-msgio/pbio"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"golang.org/x/exp/slices"

	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

func (s *FilterTestSuite) TestCreateSubscription() {
	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())
	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestModifySubscription() {

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch()), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	// Subscribe to another content_topic
	newContentTopic := "Topic_modified"
	s.subDetails = s.subscribe(s.testTopic, newContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(newContentTopic, utils.GetUnixEpoch()), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (s *FilterTestSuite) TestMultipleMessages() {

	// Initial subscribe
	s.subDetails = s.subscribe(s.testTopic, s.testContentTopic, s.fullNodeHost.ID())

	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "first"), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)

	s.waitForMsg(func() {
		_, err := s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "second"), relay.WithPubSubTopic(s.testTopic))
		s.Require().NoError(err)

	}, s.subDetails[0].C)
}

func (wf *WakuFilterLightNode) incorrectSubscribeRequest(ctx context.Context, params *FilterSubscribeParameters,
	reqType pb.FilterSubscribeRequest_FilterSubscribeType, contentFilter protocol.ContentFilter) error {

	const FilterSubscribeID_Incorrect1 = libp2pProtocol.ID("/vac/waku/filter-subscribe/abcd")

	conn, err := wf.h.NewStream(ctx, params.selectedPeer, FilterSubscribeID_Incorrect1)
	if err != nil {
		wf.metrics.RecordError(dialFailure)
		return err
	}
	defer conn.Close()

	writer := pbio.NewDelimitedWriter(conn)
	reader := pbio.NewDelimitedReader(conn, math.MaxInt32)

	request := &pb.FilterSubscribeRequest{
		RequestId:           hex.EncodeToString(params.requestID),
		FilterSubscribeType: reqType,
		PubsubTopic:         &contentFilter.PubsubTopic,
		ContentTopics:       contentFilter.ContentTopicsList(),
	}

	wf.log.Debug("sending FilterSubscribeRequest", zap.Stringer("request", request))
	err = writer.WriteMsg(request)
	if err != nil {
		wf.metrics.RecordError(writeRequestFailure)
		wf.log.Error("sending FilterSubscribeRequest", zap.Error(err))
		return err
	}

	filterSubscribeResponse := &pb.FilterSubscribeResponse{}
	err = reader.ReadMsg(filterSubscribeResponse)
	if err != nil {
		wf.log.Error("receiving FilterSubscribeResponse", zap.Error(err))
		wf.metrics.RecordError(decodeRPCFailure)
		return err
	}
	if filterSubscribeResponse.RequestId != request.RequestId {
		wf.log.Error("requestID mismatch", zap.String("expected", request.RequestId), zap.String("received", filterSubscribeResponse.RequestId))
		wf.metrics.RecordError(requestIDMismatch)
		err := NewFilterError(300, "request_id_mismatch")
		return &err
	}

	if filterSubscribeResponse.StatusCode != http.StatusOK {
		wf.metrics.RecordError(errorResponse)
		err := NewFilterError(int(filterSubscribeResponse.StatusCode), filterSubscribeResponse.GetStatusDesc())
		return &err
	}

	return nil
}

func (wf *WakuFilterLightNode) IncorrectSubscribe(ctx context.Context, contentFilter protocol.ContentFilter, opts ...FilterSubscribeOption) ([]*subscription.SubscriptionDetails, error) {
	wf.RLock()
	defer wf.RUnlock()
	if err := wf.ErrOnNotRunning(); err != nil {
		return nil, err
	}

	if len(contentFilter.ContentTopics) == 0 {
		return nil, errors.New("at least one content topic is required")
	}
	if slices.Contains[string](contentFilter.ContentTopicsList(), "") {
		return nil, errors.New("one or more content topics specified is empty")
	}

	if len(contentFilter.ContentTopics) > MaxContentTopicsPerRequest {
		return nil, fmt.Errorf("exceeds maximum content topics: %d", MaxContentTopicsPerRequest)
	}

	params := new(FilterSubscribeParameters)
	params.log = wf.log
	params.host = wf.h
	params.pm = wf.pm

	optList := DefaultSubscriptionOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		err := opt(params)
		if err != nil {
			return nil, err
		}
	}

	pubSubTopicMap, err := protocol.ContentFilterToPubSubTopicMap(contentFilter)

	if err != nil {
		return nil, err
	}
	failedContentTopics := []string{}
	subscriptions := make([]*subscription.SubscriptionDetails, 0)
	for pubSubTopic, cTopics := range pubSubTopicMap {
		var selectedPeer peer.ID
		//TO Optimize: find a peer with all pubSubTopics in the list if possible, if not only then look for single pubSubTopic
		if params.pm != nil && params.selectedPeer == "" {
			selectedPeer, err = wf.pm.SelectPeer(
				peermanager.PeerSelectionCriteria{
					SelectionType: params.peerSelectionType,
					Proto:         FilterSubscribeID_v20beta1,
					PubsubTopics:  []string{pubSubTopic},
					SpecificPeers: params.preferredPeers,
					Ctx:           ctx,
				},
			)
		} else {
			selectedPeer = params.selectedPeer
		}

		if selectedPeer == "" {
			wf.metrics.RecordError(peerNotFoundFailure)
			wf.log.Error("selecting peer", zap.String("pubSubTopic", pubSubTopic), zap.Strings("contentTopics", cTopics),
				zap.Error(err))
			failedContentTopics = append(failedContentTopics, cTopics...)
			continue
		}

		var cFilter protocol.ContentFilter
		cFilter.PubsubTopic = pubSubTopic
		cFilter.ContentTopics = protocol.NewContentTopicSet(cTopics...)

		err := wf.incorrectSubscribeRequest(ctx, params, pb.FilterSubscribeRequest_SUBSCRIBE, cFilter)
		if err != nil {
			wf.log.Error("Failed to subscribe", zap.String("pubSubTopic", pubSubTopic), zap.Strings("contentTopics", cTopics),
				zap.Error(err))
			failedContentTopics = append(failedContentTopics, cTopics...)
			continue
		}
		subscriptions = append(subscriptions, wf.subscriptions.NewSubscription(selectedPeer, cFilter))
	}

	if len(failedContentTopics) > 0 {
		return subscriptions, fmt.Errorf("subscriptions failed for contentTopics: %s", strings.Join(failedContentTopics, ","))
	} else {
		return subscriptions, nil
	}
}

func (s *FilterTestSuite) TestIncorrectSubscribeIdentifier() {
	log := utils.Logger()
	s.log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	s.ctx, s.ctxCancel = context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds

	s.testTopic = defaultTestPubSubTopic
	s.testContentTopic = defaultTestContentTopic

	s.lightNode = s.makeWakuFilterLightNode(true, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, false, true)

	//Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)

	// Subscribe with incorrect SubscribeID
	s.contentFilter = protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}
	_, err := s.lightNode.IncorrectSubscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().Error(err)

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}

func (wf *WakuFilterLightNode) startWithIncorrectPushProto() error {
	const FilterPushID_Incorrect1 = libp2pProtocol.ID("/vac/waku/filter-push/abcd")

	wf.subscriptions = subscription.NewSubscriptionMap(wf.log)
	wf.h.SetStreamHandlerMatch(FilterPushID_v20beta1, protocol.PrefixTextMatch(string(FilterPushID_Incorrect1)), wf.onRequest(wf.Context()))

	wf.log.Info("filter-push incorrect protocol started")
	return nil
}

func (s *FilterTestSuite) TestIncorrectPushIdentifier() {
	log := utils.Logger()
	s.log = log
	s.wg = &sync.WaitGroup{}

	// Create test context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Test can't exceed 10 seconds
	s.ctx = ctx
	s.ctxCancel = cancel

	s.testTopic = defaultTestPubSubTopic
	s.testContentTopic = defaultTestContentTopic

	s.lightNode = s.makeWakuFilterLightNode(false, true)

	s.relayNode, s.fullNode = s.makeWakuFilterFullNode(s.testTopic, false, true)

	// Re-start light node with unsupported prefix for match func
	s.lightNode.Stop()
	err := s.lightNode.CommonService.Start(s.ctx, s.lightNode.startWithIncorrectPushProto)
	s.Require().NoError(err)

	// Connect nodes
	s.lightNodeHost.Peerstore().AddAddr(s.fullNodeHost.ID(), tests.GetHostAddress(s.fullNodeHost), peerstore.PermanentAddrTTL)
	err = s.lightNodeHost.Peerstore().AddProtocols(s.fullNodeHost.ID(), FilterSubscribeID_v20beta1)
	s.Require().NoError(err)

	// Subscribe
	s.contentFilter = protocol.ContentFilter{PubsubTopic: s.testTopic, ContentTopics: protocol.NewContentTopicSet(s.testContentTopic)}
	s.subDetails, err = s.lightNode.Subscribe(s.ctx, s.contentFilter, WithPeer(s.fullNodeHost.ID()))
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	// Send message
	_, err = s.relayNode.Publish(s.ctx, tests.CreateWakuMessage(s.testContentTopic, utils.GetUnixEpoch(), "second"), relay.WithPubSubTopic(s.testTopic))
	s.Require().NoError(err)

	// Message should never arrive -> exit after timeout
	select {
	case msg := <-s.subDetails[0].C:
		s.log.Info("Light node received a msg")
		s.Require().Nil(msg)
	case <-time.After(1 * time.Second):
		s.Require().True(true)
	}

	_, err = s.lightNode.UnsubscribeAll(s.ctx)
	s.Require().NoError(err)
}
