// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package rpc

import (
	"context"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/tikvpb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/metrics"
	"google.golang.org/grpc"
	gcodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	gstatus "google.golang.org/grpc/status"
)

// Client is a client that sends RPC.
// It should not be used after calling Close().
type Client interface {
	// Close should release all data.
	Close() error
	// SendRequest sends Request.
	SendRequest(ctx context.Context, addr string, req *Request, timeout time.Duration) (*Response, error)
}

type connArray struct {
	conf  *config.RPC
	index uint32
	conns []*grpc.ClientConn
	// Bind with a background goroutine to process coprocessor streaming timeout.
	streamTimeout chan *Lease

	// For batch commands.
	batchCommandsCh      chan *batchCommandsEntry
	batchCommandsClients []*batchCommandsClient
	transportLayerLoad   uint64
}

type batchCommandsClient struct {
	conf               *config.Batch
	conn               *grpc.ClientConn
	client             tikvpb.Tikv_BatchCommandsClient
	batched            sync.Map
	idAlloc            uint64
	transportLayerLoad *uint64

	// Indicates the batch client is closed explicitly or not.
	closed int32
	// Protect client when re-create the streaming.
	clientLock sync.Mutex
}

func (c *batchCommandsClient) isStopped() bool {
	return atomic.LoadInt32(&c.closed) != 0
}

func (c *batchCommandsClient) failPendingRequests(err error) {
	c.batched.Range(func(key, value interface{}) bool {
		id, _ := key.(uint64)
		entry, _ := value.(*batchCommandsEntry)
		entry.err = err
		close(entry.res)
		c.batched.Delete(id)
		return true
	})
}

func (c *batchCommandsClient) batchRecvLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("batchRecvLoop %v", r)
			log.Infof("Restart batchRecvLoop")
			go c.batchRecvLoop()
		}
	}()

	for {
		// When `conn.Close()` is called, `client.Recv()` will return an error.
		resp, err := c.client.Recv()
		if err != nil {
			if c.isStopped() {
				return
			}
			log.Errorf("batchRecvLoop error when receive: %v", err)

			// Hold the lock to forbid batchSendLoop using the old client.
			c.clientLock.Lock()
			c.failPendingRequests(err) // fail all pending requests.
			for {                      // try to re-create the streaming in the loop.
				// Re-establish a application layer stream. TCP layer is handled by gRPC.
				tikvClient := tikvpb.NewTikvClient(c.conn)
				streamClient, err := tikvClient.BatchCommands(context.TODO())
				if err == nil {
					log.Infof("batchRecvLoop re-create streaming success")
					c.client = streamClient
					break
				}
				log.Errorf("batchRecvLoop re-create streaming fail: %v", err)
				// TODO: Use a more smart backoff strategy.
				time.Sleep(time.Second)
			}
			c.clientLock.Unlock()
			continue
		}

		responses := resp.GetResponses()
		for i, requestID := range resp.GetRequestIds() {
			value, ok := c.batched.Load(requestID)
			if !ok {
				// There shouldn't be any unknown responses because if the old entries
				// are cleaned by `failPendingRequests`, the stream must be re-created
				// so that old responses will be never received.
				panic("batchRecvLoop receives a unknown response")
			}
			entry := value.(*batchCommandsEntry)
			if atomic.LoadInt32(&entry.canceled) == 0 {
				// Put the response only if the request is not canceled.
				entry.res <- responses[i]
			}
			c.batched.Delete(requestID)
		}

		transportLayerLoad := resp.GetTransportLayerLoad()
		if transportLayerLoad > 0.0 && c.conf.MaxWaitTime > 0 {
			// We need to consider TiKV load only if batch-wait strategy is enabled.
			atomic.StoreUint64(c.transportLayerLoad, transportLayerLoad)
		}
	}
}

func newConnArray(addr string, conf *config.RPC) (*connArray, error) {
	a := &connArray{
		conf:                 conf,
		index:                0,
		conns:                make([]*grpc.ClientConn, conf.MaxConnectionCount),
		streamTimeout:        make(chan *Lease, 1024),
		batchCommandsCh:      make(chan *batchCommandsEntry, conf.Batch.MaxBatchSize),
		batchCommandsClients: make([]*batchCommandsClient, 0, conf.Batch.MaxBatchSize),
		transportLayerLoad:   0,
	}
	if err := a.Init(addr); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *connArray) Init(addr string) error {
	opt := grpc.WithInsecure()
	if len(a.conf.Security.SSLCA) != 0 {
		tlsConfig, err := a.conf.Security.ToTLSConfig()
		if err != nil {
			return err
		}
		opt = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}

	unaryInterceptor := grpc_prometheus.UnaryClientInterceptor
	streamInterceptor := grpc_prometheus.StreamClientInterceptor
	if a.conf.EnableOpenTracing {
		unaryInterceptor = grpc_middleware.ChainUnaryClient(
			unaryInterceptor,
			grpc_opentracing.UnaryClientInterceptor(),
		)
		streamInterceptor = grpc_middleware.ChainStreamClient(
			streamInterceptor,
			grpc_opentracing.StreamClientInterceptor(),
		)
	}

	allowBatch := a.conf.Batch.MaxBatchSize > 0
	for i := range a.conns {
		ctx, cancel := context.WithTimeout(context.Background(), a.conf.DialTimeout)
		conn, err := grpc.DialContext(
			ctx,
			addr,
			opt,
			grpc.WithInitialWindowSize(int32(a.conf.GrpcInitialWindowSize)),
			grpc.WithInitialConnWindowSize(int32(a.conf.GrpcInitialConnWindowSize)),
			grpc.WithUnaryInterceptor(unaryInterceptor),
			grpc.WithStreamInterceptor(streamInterceptor),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(a.conf.GrpcMaxCallMsgSize)),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(a.conf.GrpcMaxSendMsgSize)),
			grpc.WithBackoffMaxDelay(time.Second*3),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                a.conf.GrpcKeepAliveTime,
				Timeout:             a.conf.GrpcKeepAliveTimeout,
				PermitWithoutStream: true,
			}),
		)
		cancel()
		if err != nil {
			// Cleanup if the initialization fails.
			a.Close()
			return errors.WithStack(err)
		}
		a.conns[i] = conn

		if allowBatch {
			// Initialize batch streaming clients.
			tikvClient := tikvpb.NewTikvClient(conn)
			streamClient, err := tikvClient.BatchCommands(context.TODO())
			if err != nil {
				a.Close()
				return errors.WithStack(err)
			}
			batchClient := &batchCommandsClient{
				conf:               &a.conf.Batch,
				conn:               conn,
				client:             streamClient,
				batched:            sync.Map{},
				idAlloc:            0,
				transportLayerLoad: &a.transportLayerLoad,
				closed:             0,
			}
			a.batchCommandsClients = append(a.batchCommandsClients, batchClient)
			go batchClient.batchRecvLoop()
		}
	}
	go CheckStreamTimeoutLoop(a.streamTimeout)
	if allowBatch {
		go a.batchSendLoop()
	}

	return nil
}

func (a *connArray) Get() *grpc.ClientConn {
	next := atomic.AddUint32(&a.index, 1) % uint32(len(a.conns))
	return a.conns[next]
}

func (a *connArray) Close() {
	// Close all batchRecvLoop.
	for _, c := range a.batchCommandsClients {
		// After connections are closed, `batchRecvLoop`s will check the flag.
		atomic.StoreInt32(&c.closed, 1)
	}
	close(a.batchCommandsCh)
	for i, c := range a.conns {
		if c != nil {
			c.Close()
			a.conns[i] = nil
		}
	}
	close(a.streamTimeout)
}

type batchCommandsEntry struct {
	req *tikvpb.BatchCommandsRequest_Request
	res chan *tikvpb.BatchCommandsResponse_Response

	// Indicated the request is canceled or not.
	canceled int32
	err      error
}

// fetchAllPendingRequests fetches all pending requests from the channel.
func fetchAllPendingRequests(
	ch chan *batchCommandsEntry,
	maxBatchSize int,
	entries *[]*batchCommandsEntry,
	requests *[]*tikvpb.BatchCommandsRequest_Request,
) {
	// Block on the first element.
	headEntry := <-ch
	if headEntry == nil {
		return
	}
	*entries = append(*entries, headEntry)
	*requests = append(*requests, headEntry.req)

	// This loop is for trying best to collect more requests.
	for len(*entries) < maxBatchSize {
		select {
		case entry := <-ch:
			if entry == nil {
				return
			}
			*entries = append(*entries, entry)
			*requests = append(*requests, entry.req)
		default:
			return
		}
	}
}

// fetchMorePendingRequests fetches more pending requests from the channel.
func fetchMorePendingRequests(
	ch chan *batchCommandsEntry,
	maxBatchSize int,
	batchWaitSize int,
	maxWaitTime time.Duration,
	entries *[]*batchCommandsEntry,
	requests *[]*tikvpb.BatchCommandsRequest_Request,
) {
	waitStart := time.Now()

	// Try to collect `batchWaitSize` requests, or wait `maxWaitTime`.
	after := time.NewTimer(maxWaitTime)
	for len(*entries) < batchWaitSize {
		select {
		case entry := <-ch:
			if entry == nil {
				return
			}
			*entries = append(*entries, entry)
			*requests = append(*requests, entry.req)
		case waitEnd := <-after.C:
			metrics.BatchWaitDuration.Observe(float64(waitEnd.Sub(waitStart)))
			return
		}
	}
	after.Stop()

	// Do an additional non-block try.
	for len(*entries) < maxBatchSize {
		select {
		case entry := <-ch:
			if entry == nil {
				return
			}
			*entries = append(*entries, entry)
			*requests = append(*requests, entry.req)
		default:
			metrics.BatchWaitDuration.Observe(float64(time.Since(waitStart)))
			return
		}
	}
}

func (a *connArray) batchSendLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("batchSendLoop %v", r)
			log.Infof("Restart batchSendLoop")
			go a.batchSendLoop()
		}
	}()

	conf := &a.conf.Batch

	entries := make([]*batchCommandsEntry, 0, conf.MaxBatchSize)
	requests := make([]*tikvpb.BatchCommandsRequest_Request, 0, conf.MaxBatchSize)
	requestIDs := make([]uint64, 0, conf.MaxBatchSize)

	for {
		// Choose a connection by round-robbin.
		next := atomic.AddUint32(&a.index, 1) % uint32(len(a.conns))
		batchCommandsClient := a.batchCommandsClients[next]

		entries = entries[:0]
		requests = requests[:0]
		requestIDs = requestIDs[:0]

		metrics.PendingBatchRequests.Set(float64(len(a.batchCommandsCh)))
		fetchAllPendingRequests(a.batchCommandsCh, int(conf.MaxBatchSize), &entries, &requests)

		if len(entries) < int(conf.MaxBatchSize) && conf.MaxWaitTime > 0 {
			transportLayerLoad := atomic.LoadUint64(batchCommandsClient.transportLayerLoad)
			// If the target TiKV is overload, wait a while to collect more requests.
			if uint(transportLayerLoad) >= conf.OverloadThreshold {
				fetchMorePendingRequests(
					a.batchCommandsCh, int(conf.MaxBatchSize), int(conf.MaxWaitSize),
					conf.MaxWaitTime, &entries, &requests,
				)
			}
		}

		length := len(requests)
		maxBatchID := atomic.AddUint64(&batchCommandsClient.idAlloc, uint64(length))
		for i := 0; i < length; i++ {
			requestID := uint64(i) + maxBatchID - uint64(length)
			requestIDs = append(requestIDs, requestID)
		}

		request := &tikvpb.BatchCommandsRequest{
			Requests:   requests,
			RequestIds: requestIDs,
		}

		// Use the lock to protect the stream client won't be replaced by RecvLoop,
		// and new added request won't be removed by `failPendingRequests`.
		batchCommandsClient.clientLock.Lock()
		for i, requestID := range request.RequestIds {
			batchCommandsClient.batched.Store(requestID, entries[i])
		}
		err := batchCommandsClient.client.Send(request)
		batchCommandsClient.clientLock.Unlock()
		if err != nil {
			log.Errorf("batch commands send error: %v", err)
			batchCommandsClient.failPendingRequests(err)
		}
	}
}

// rpcClient is RPC client struct.
// TODO: Add flow control between RPC clients in TiDB ond RPC servers in TiKV.
// Since we use shared client connection to communicate to the same TiKV, it's possible
// that there are too many concurrent requests which overload the service of TiKV.
// TODO: Implement background cleanup. It adds a background goroutine to periodically check
// whether there is any connection is idle and then close and remove these idle connections.
type rpcClient struct {
	sync.RWMutex
	isClosed bool
	conns    map[string]*connArray
	conf     *config.RPC
}

// NewRPCClient manages connections and rpc calls with tikv-servers.
func NewRPCClient(conf *config.RPC) Client {
	return &rpcClient{
		conns: make(map[string]*connArray),
		conf:  conf,
	}
}

func (c *rpcClient) getConnArray(addr string) (*connArray, error) {
	c.RLock()
	if c.isClosed {
		c.RUnlock()
		return nil, errors.Errorf("rpcClient is closed")
	}
	array, ok := c.conns[addr]
	c.RUnlock()
	if !ok {
		var err error
		array, err = c.createConnArray(addr)
		if err != nil {
			return nil, err
		}
	}
	return array, nil
}

func (c *rpcClient) createConnArray(addr string) (*connArray, error) {
	c.Lock()
	defer c.Unlock()
	array, ok := c.conns[addr]
	if !ok {
		var err error
		array, err = newConnArray(addr, c.conf)
		if err != nil {
			return nil, err
		}
		c.conns[addr] = array
	}
	return array, nil
}

func (c *rpcClient) closeConns() {
	c.Lock()
	if !c.isClosed {
		c.isClosed = true
		// close all connections
		for _, array := range c.conns {
			array.Close()
		}
	}
	c.Unlock()
}

func sendBatchRequest(
	ctx context.Context,
	addr string,
	connArray *connArray,
	req *tikvpb.BatchCommandsRequest_Request,
	timeout time.Duration,
) (*Response, error) {
	entry := &batchCommandsEntry{
		req:      req,
		res:      make(chan *tikvpb.BatchCommandsResponse_Response, 1),
		canceled: 0,
		err:      nil,
	}
	ctx1, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case connArray.batchCommandsCh <- entry:
	case <-ctx1.Done():
		log.Warnf("SendRequest to %s is timeout", addr)
		return nil, errors.WithStack(gstatus.Error(gcodes.DeadlineExceeded, "Canceled or timeout"))
	}

	select {
	case res, ok := <-entry.res:
		if !ok {
			return nil, errors.WithStack(entry.err)
		}
		return FromBatchCommandsResponse(res), nil
	case <-ctx1.Done():
		atomic.StoreInt32(&entry.canceled, 1)
		log.Warnf("SendRequest to %s is canceled", addr)
		return nil, errors.WithStack(gstatus.Error(gcodes.DeadlineExceeded, "Canceled or timeout"))
	}
}

// SendRequest sends a Request to server and receives Response.
func (c *rpcClient) SendRequest(ctx context.Context, addr string, req *Request, timeout time.Duration) (*Response, error) {
	start := time.Now()
	reqType := req.Type.String()
	storeID := strconv.FormatUint(req.Context.GetPeer().GetStoreId(), 10)
	defer func() {
		metrics.SendReqHistogram.WithLabelValues(reqType, storeID).Observe(time.Since(start).Seconds())
	}()

	connArray, err := c.getConnArray(addr)
	if err != nil {
		return nil, err
	}

	if c.conf.Batch.MaxBatchSize > 0 {
		if batchReq := req.ToBatchCommandsRequest(); batchReq != nil {
			return sendBatchRequest(ctx, addr, connArray, batchReq, timeout)
		}
	}

	client := tikvpb.NewTikvClient(connArray.Get())

	if req.Type != CmdCopStream {
		ctx1, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return CallRPC(ctx1, client, req)
	}

	// Coprocessor streaming request.
	// Use context to support timeout for grpc streaming client.
	ctx1, cancel := context.WithCancel(ctx)
	defer cancel()
	resp, err := CallRPC(ctx1, client, req)
	if err != nil {
		return nil, err
	}

	// Put the lease object to the timeout channel, so it would be checked periodically.
	copStream := resp.CopStream
	copStream.Timeout = timeout
	copStream.Lease.Cancel = cancel
	connArray.streamTimeout <- &copStream.Lease

	// Read the first streaming response to get CopStreamResponse.
	// This can make error handling much easier, because SendReq() retry on
	// region error automatically.
	var first *coprocessor.Response
	first, err = copStream.Recv()
	if err != nil {
		if errors.Cause(err) != io.EOF {
			return nil, err
		}
		log.Debug("copstream returns nothing for the request.")
	}
	copStream.Response = first
	return resp, nil
}

func (c *rpcClient) Close() error {
	c.closeConns()
	return nil
}
