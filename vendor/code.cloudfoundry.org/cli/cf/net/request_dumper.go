package net

import (
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
)

//go:generate counterfeiter . RequestDumperInterface

type RequestDumperInterface interface {
	DumpRequest(*http.Request)
	DumpResponse(*http.Response)
}

type RequestDumper struct {
	printer trace.Printer
}

func NewRequestDumper(printer trace.Printer) RequestDumper {
	return RequestDumper{printer: printer}
}

func (p RequestDumper) DumpRequest(req *http.Request) {
	shouldDisplayBody := !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data")
	dumpedRequest, err := httputil.DumpRequest(req, shouldDisplayBody)
	if err != nil {
		p.printer.Printf(T("Error dumping request\n{{.Err}}\n", map[string]interface{}{"Err": err}))
	} else {
		p.printer.Printf("\n%s [%s]\n%s\n", terminal.HeaderColor(T("REQUEST:")), time.Now().Format(time.RFC3339), trace.Sanitize(string(dumpedRequest)))
		if !shouldDisplayBody {
			p.printer.Println(T("[MULTIPART/FORM-DATA CONTENT HIDDEN]"))
		}
	}
}

func (p RequestDumper) DumpResponse(res *http.Response) {
	dumpedResponse, err := httputil.DumpResponse(res, true)
	if err != nil {
		p.printer.Printf(T("Error dumping response\n{{.Err}}\n", map[string]interface{}{"Err": err}))
	} else {
		p.printer.Printf("\n%s [%s]\n%s\n", terminal.HeaderColor(T("RESPONSE:")), time.Now().Format(time.RFC3339), trace.Sanitize(string(dumpedResponse)))
	}
}
