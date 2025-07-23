package debugger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/google/go-dap"
	"github.com/hashicorp/terraform/internal/moduletest/graph"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ErrServerStopped = errors.New("dap: server stopped")

type DapMsg struct {
	msg dap.Message
}

type Server struct {
	sess *DebugSession

	mu sync.RWMutex
	ch chan DapMsg

	eg     *errgroup.Group
	ctx    context.Context
	cancel context.CancelCauseFunc

	initialized bool

	conn net.Conn
}

func NewServer(ctx *graph.DebugContext, dir string) *Server {
	sess := &DebugSession{
		Step:    1,
		State:   make(map[string]any),
		Context: ctx,
	}
	sess.WriteCh = make(chan DapMsg)
	return &Server{sess: sess}
}

type Listener net.Listener

func (s *Server) Serve(ctx context.Context, ls Listener) error {
	s.ch = s.sess.WriteCh
	s.ctx, s.cancel = context.WithCancelCause(ctx)

	conn, err := ls.Accept()
	if err != nil {
		return err
	}
	s.conn = conn

	// Start an error group to handle server-initiated tasks.
	s.eg, _ = errgroup.WithContext(s.ctx)
	s.eg.Go(func() error {
		<-s.ctx.Done()
		return s.ctx.Err()
	})

	// handle incoming requests
	eg, _ := errgroup.WithContext(s.ctx)
	eg.Go(func() error {
		err := s.HandleDAPRequests()
		return err
	})

	eg, _ = errgroup.WithContext(s.ctx)
	eg.Go(func() error {
		err := s.handleWriteRequests()
		return err
	})

	// wait for the server-initialized tasks to finish
	eg.Go(func() error {
		err := s.eg.Wait()
		return err
	})

	return eg.Wait()
}

func (s *Server) HandleDAPRequests() error {
	conn := s.conn
	fmt.Printf("New connection from: %s\n", conn.RemoteAddr())
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		msg, err := dap.ReadProtocolMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("client disconnected")
				return nil
			}
			fmt.Printf("error reading message: %v\n", err)
			return err
		}

		if _, ok := msg.(*dap.DisconnectRequest); ok {
			s.cancel(ErrServerStopped)
			// terminate already shut things down and triggered stopping
			return nil
		}

		switch msg := msg.(type) {
		case dap.RequestMessage:
			s.eg.Go(func() error {
				_, err := s.handleMessage(s.ctx, msg)
				return err
			})
		default:
			fmt.Printf("Non-request message type: %T", msg)
		}
	}
}

func (s *Server) handleMessage(c context.Context, m dap.Message) (dap.ResponseMessage, error) {
	switch req := m.(type) {
	case *dap.InitializeRequest:
		if s.initialized {
			return nil, errors.New("already initialized")
		}

		resp, err := s.sess.Initialize(c, req)
		if err != nil {
			return nil, err
		}
		s.initialized = true
		return resp, nil
	case *dap.LaunchRequest:
		return s.sess.Launch(c, req)
	case *dap.AttachRequest:
		return nil, errors.New("not implemented")
	case *dap.SetBreakpointsRequest:
		return s.sess.SetBreakpoints(c, req)
	case *dap.SetExceptionBreakpointsRequest:
		resp := &dap.SetExceptionBreakpointsResponse{}
		resp.Success = true
		s.sess.send(req, resp)
		return resp, nil
	case *dap.BreakpointLocationsRequest:
		return s.sess.BreakpointLocations(c, req)
	case *dap.SetFunctionBreakpointsRequest:
		resp := &dap.SetFunctionBreakpointsResponse{}
		resp.Success = true
		s.sess.send(req, resp)
		return resp, nil
	case *dap.SetInstructionBreakpointsRequest:
		resp := &dap.SetInstructionBreakpointsResponse{}
		resp.Success = true
		s.sess.send(req, resp)
		return resp, nil
	case *dap.SetDataBreakpointsRequest:
		resp := &dap.SetDataBreakpointsResponse{}
		resp.Success = true
		s.sess.send(req, resp)
		return resp, nil
	case *dap.ConfigurationDoneRequest:
		fmt.Println("Configuration done")
		resp, err := s.sess.ConfigurationDone(c, req)
		s.sess.send(req, resp)
		return resp, err
	case *dap.DisconnectRequest:
		return s.sess.Disconnect(c, req)
	case *dap.TerminateRequest:
		return s.sess.Terminate(c, req)
	case *dap.ContinueRequest:
		return s.sess.Continue(c, req)
	case *dap.NextRequest:
		return s.sess.Next(c, req)
	// case *dap.RestartRequest:
	// 	return s.sess.Restart.Do(c, req)
	case *dap.ThreadsRequest:
		return s.sess.Threads(c, req)
	case *dap.StackTraceRequest:
		return s.sess.StackTrace(c, req)
	// case *dap.PauseRequest:
	// 	return s.sess.StackTrace(c, req)
	case *dap.EvaluateRequest:
		return s.sess.Evaluate(c, req)
	case *dap.SourceRequest:
		return s.sess.SourceReq(c, req)
	case *dap.VariablesRequest:
		return s.sess.Variables(c, req)
	case *dap.ScopesRequest:
		return s.sess.Scopes(c, req)
	default:
		return nil, errors.New("not implemented")
	}
}

func (s *Server) handleWriteRequests() error {
	var seq int
	for m := range s.ch {
		switch m := m.msg.(type) {
		case dap.RequestMessage:
			m.GetRequest().Seq = seq
			m.GetRequest().Type = "request"
		case dap.EventMessage:
			m.GetEvent().Seq = seq
			m.GetEvent().Type = "event"
		case dap.ResponseMessage:
			m.GetResponse().Seq = seq
			m.GetResponse().Type = "response"
		}
		seq++

		if err := dap.WriteProtocolMessage(s.conn, m.msg); err != nil {
			return err
		}
	}
	return nil
}
