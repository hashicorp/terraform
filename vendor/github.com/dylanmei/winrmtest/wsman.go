package winrmtest

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/antchfx/xquery/xml"
	"github.com/satori/go.uuid"
)

type wsman struct {
	commands     []*command
	identitySeed int
}

type command struct {
	id      string
	matcher MatcherFunc
	handler CommandFunc
}

func (w *wsman) HandleCommand(m MatcherFunc, f CommandFunc) string {
	id := uuid.NewV4().String()
	w.commands = append(w.commands, &command{
		id:      id,
		matcher: m,
		handler: f,
	})

	return id
}

func (w *wsman) CommandByText(cmd string) *command {
	for _, c := range w.commands {
		if c.matcher(cmd) {
			return c
		}
	}
	return nil
}

func (w *wsman) CommandByID(id string) *command {
	for _, c := range w.commands {
		if c.id == id {
			return c
		}
	}
	return nil
}

func (w *wsman) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/soap+xml")

	defer r.Body.Close()
	env, err := xmlquery.Parse(r.Body)

	if err != nil {
		return
	}

	action := readAction(env)
	switch {
	case strings.HasSuffix(action, "transfer/Create"):
		// create a new shell

		rw.Write([]byte(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:ShellId>123</rsp:ShellId>
			</env:Envelope>`))

	case strings.HasSuffix(action, "shell/Command"):
		// execute on behalf of the client
		text := readCommand(env)
		cmd := w.CommandByText(text)

		if cmd == nil {
			fmt.Printf("I don't know this command: Command=%s\n", text)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write([]byte(fmt.Sprintf(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:CommandId>%s</rsp:CommandId>
			</env:Envelope>`, cmd.id)))

	case strings.HasSuffix(action, "shell/Receive"):
		// client ready to receive the results

		id := readCommandIDFromDesiredStream(env)
		cmd := w.CommandByID(id)

		if cmd == nil {
			fmt.Printf("I don't know this command: CommandId=%s\n", id)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		result := cmd.handler(stdout, stderr)
		content := base64.StdEncoding.EncodeToString(stdout.Bytes())

		rw.Write([]byte(fmt.Sprintf(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:ReceiveResponse>
					<rsp:Stream Name="stdout" CommandId="%s">%s</rsp:Stream>
					<rsp:Stream Name="stdout" CommandId="%s" End="true"></rsp:Stream>
					<rsp:Stream Name="stderr" CommandId="%s" End="true"></rsp:Stream>
					<rsp:CommandState State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done">
						<rsp:ExitCode>%d</rsp:ExitCode>
					</rsp:CommandState>
				</rsp:ReceiveResponse>
			</env:Envelope>`, id, content, id, id, result)))

	case strings.HasSuffix(action, "shell/Signal"):
		// end of the shell command
		rw.WriteHeader(http.StatusOK)
	case strings.HasSuffix(action, "transfer/Delete"):
		// end of the session
		rw.WriteHeader(http.StatusOK)
	default:
		fmt.Printf("I don't know this action: %s\n", action)
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func readAction(env *xmlquery.Node) string {
	xpath := xmlquery.FindOne(env, "//a:Action")
	if xpath == nil {
		return ""
	}

	return xpath.InnerText()
}

func readCommand(env *xmlquery.Node) string {
	xpath := xmlquery.FindOne(env, "//rsp:Command")
	if xpath == nil {
		return ""
	}

	if unquoted, err := strconv.Unquote(xpath.InnerText()); err == nil {
		return unquoted
	}
	return xpath.InnerText()
}

func readCommandIDFromDesiredStream(env *xmlquery.Node) string {
	xpath := xmlquery.FindOne(env, "//rsp:DesiredStream")
	if xpath == nil {
		return ""
	}

	return xpath.SelectAttr("CommandId")
}
