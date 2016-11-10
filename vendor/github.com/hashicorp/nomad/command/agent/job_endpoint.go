package agent

import (
	"net/http"
	"strings"

	"github.com/hashicorp/nomad/nomad/structs"
)

func (s *HTTPServer) JobsRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	switch req.Method {
	case "GET":
		return s.jobListRequest(resp, req)
	case "PUT", "POST":
		return s.jobUpdate(resp, req, "")
	default:
		return nil, CodedError(405, ErrInvalidMethod)
	}
}

func (s *HTTPServer) jobListRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	args := structs.JobListRequest{}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.JobListResponse
	if err := s.agent.RPC("Job.List", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Jobs == nil {
		out.Jobs = make([]*structs.JobListStub, 0)
	}
	return out.Jobs, nil
}

func (s *HTTPServer) JobSpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/job/")
	switch {
	case strings.HasSuffix(path, "/evaluate"):
		jobName := strings.TrimSuffix(path, "/evaluate")
		return s.jobForceEvaluate(resp, req, jobName)
	case strings.HasSuffix(path, "/allocations"):
		jobName := strings.TrimSuffix(path, "/allocations")
		return s.jobAllocations(resp, req, jobName)
	case strings.HasSuffix(path, "/evaluations"):
		jobName := strings.TrimSuffix(path, "/evaluations")
		return s.jobEvaluations(resp, req, jobName)
	case strings.HasSuffix(path, "/periodic/force"):
		jobName := strings.TrimSuffix(path, "/periodic/force")
		return s.periodicForceRequest(resp, req, jobName)
	case strings.HasSuffix(path, "/plan"):
		jobName := strings.TrimSuffix(path, "/plan")
		return s.jobPlan(resp, req, jobName)
	case strings.HasSuffix(path, "/summary"):
		jobName := strings.TrimSuffix(path, "/summary")
		return s.jobSummaryRequest(resp, req, jobName)
	default:
		return s.jobCRUD(resp, req, path)
	}
}

func (s *HTTPServer) jobForceEvaluate(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	if req.Method != "PUT" && req.Method != "POST" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	args := structs.JobEvaluateRequest{
		JobID: jobName,
	}
	s.parseRegion(req, &args.Region)

	var out structs.JobRegisterResponse
	if err := s.agent.RPC("Job.Evaluate", &args, &out); err != nil {
		return nil, err
	}
	setIndex(resp, out.Index)
	return out, nil
}

func (s *HTTPServer) jobPlan(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	if req.Method != "PUT" && req.Method != "POST" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	var args structs.JobPlanRequest
	if err := decodeBody(req, &args); err != nil {
		return nil, CodedError(400, err.Error())
	}
	if args.Job == nil {
		return nil, CodedError(400, "Job must be specified")
	}
	if jobName != "" && args.Job.ID != jobName {
		return nil, CodedError(400, "Job ID does not match")
	}
	s.parseRegion(req, &args.Region)

	var out structs.JobPlanResponse
	if err := s.agent.RPC("Job.Plan", &args, &out); err != nil {
		return nil, err
	}
	setIndex(resp, out.Index)
	return out, nil
}

func (s *HTTPServer) periodicForceRequest(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	if req.Method != "PUT" && req.Method != "POST" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.PeriodicForceRequest{
		JobID: jobName,
	}
	s.parseRegion(req, &args.Region)

	var out structs.PeriodicForceResponse
	if err := s.agent.RPC("Periodic.Force", &args, &out); err != nil {
		return nil, err
	}
	setIndex(resp, out.Index)
	return out, nil
}

func (s *HTTPServer) jobAllocations(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	args := structs.JobSpecificRequest{
		JobID: jobName,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.JobAllocationsResponse
	if err := s.agent.RPC("Job.Allocations", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Allocations == nil {
		out.Allocations = make([]*structs.AllocListStub, 0)
	}
	return out.Allocations, nil
}

func (s *HTTPServer) jobEvaluations(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	args := structs.JobSpecificRequest{
		JobID: jobName,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.JobEvaluationsResponse
	if err := s.agent.RPC("Job.Evaluations", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Evaluations == nil {
		out.Evaluations = make([]*structs.Evaluation, 0)
	}
	return out.Evaluations, nil
}

func (s *HTTPServer) jobCRUD(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	switch req.Method {
	case "GET":
		return s.jobQuery(resp, req, jobName)
	case "PUT", "POST":
		return s.jobUpdate(resp, req, jobName)
	case "DELETE":
		return s.jobDelete(resp, req, jobName)
	default:
		return nil, CodedError(405, ErrInvalidMethod)
	}
}

func (s *HTTPServer) jobQuery(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	args := structs.JobSpecificRequest{
		JobID: jobName,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.SingleJobResponse
	if err := s.agent.RPC("Job.GetJob", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Job == nil {
		return nil, CodedError(404, "job not found")
	}
	return out.Job, nil
}

func (s *HTTPServer) jobUpdate(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	var args structs.JobRegisterRequest
	if err := decodeBody(req, &args); err != nil {
		return nil, CodedError(400, err.Error())
	}
	if args.Job == nil {
		return nil, CodedError(400, "Job must be specified")
	}
	if jobName != "" && args.Job.ID != jobName {
		return nil, CodedError(400, "Job ID does not match")
	}
	s.parseRegion(req, &args.Region)

	var out structs.JobRegisterResponse
	if err := s.agent.RPC("Job.Register", &args, &out); err != nil {
		return nil, err
	}
	setIndex(resp, out.Index)
	return out, nil
}

func (s *HTTPServer) jobDelete(resp http.ResponseWriter, req *http.Request,
	jobName string) (interface{}, error) {
	args := structs.JobDeregisterRequest{
		JobID: jobName,
	}
	s.parseRegion(req, &args.Region)

	var out structs.JobDeregisterResponse
	if err := s.agent.RPC("Job.Deregister", &args, &out); err != nil {
		return nil, err
	}
	setIndex(resp, out.Index)
	return out, nil
}

func (s *HTTPServer) jobSummaryRequest(resp http.ResponseWriter, req *http.Request, name string) (interface{}, error) {
	args := structs.JobSummaryRequest{
		JobID: name,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.JobSummaryResponse
	if err := s.agent.RPC("Job.Summary", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.JobSummary == nil {
		return nil, CodedError(404, "job not found")
	}
	setIndex(resp, out.Index)
	return out.JobSummary, nil
}
