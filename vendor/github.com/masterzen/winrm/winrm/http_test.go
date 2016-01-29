package winrm

import (
	"net"
	"net/http"
	"net/http/httptest"

	. "gopkg.in/check.v1"
)

var response = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:x="http://schemas.xmlsoap.org/ws/2004/09/transfer" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd">
<s:Header>
    <a:Action>http://schemas.xmlsoap.org/ws/2004/09/transfer/CreateResponse</a:Action>
    <a:MessageID>uuid:195078CF-804B-41F7-A246-9CB3C1A41A9A</a:MessageID>
    <a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To>
    <a:RelatesTo>uuid:D00059E8-57D6-4035-AD8D-3EDC495DA163</a:RelatesTo>
</s:Header>
<s:Body>
    <x:ResourceCreated>
        <a:Address>http://107.20.128.235:15985/wsman</a:Address>
        <a:ReferenceParameters>
            <w:ResourceURI>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd</w:ResourceURI>
            <w:SelectorSet>
                <w:Selector Name="ShellId">67A74734-DD32-4F10-89DE-49A060483810</w:Selector>
            </w:SelectorSet>
        </a:ReferenceParameters>
    </x:ResourceCreated>
    <rsp:Shell xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
        <rsp:ShellId>67A74734-DD32-4F10-89DE-49A060483810</rsp:ShellId>
        <rsp:ResourceUri>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd</rsp:ResourceUri>
        <rsp:Owner>Administrator</rsp:Owner>
        <rsp:ClientIP>213.41.177.193</rsp:ClientIP>
        <rsp:IdleTimeOut>PT7200.000S</rsp:IdleTimeOut>
        <rsp:InputStreams>stdin</rsp:InputStreams>
        <rsp:OutputStreams>stdout
stderr</rsp:OutputStreams>
        <rsp:ShellRunTime>P0DT0H0M1S</rsp:ShellRunTime>
        <rsp:ShellInactivity>P0DT0H0M1S</rsp:ShellInactivity>
    </rsp:Shell>
</s:Body>
</s:Envelope>`

func (s *WinRMSuite) TestHttpRequest(c *C) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		w.Write([]byte(response))
	}))
	l, err := net.Listen("tcp", "127.0.0.1:15985")
	if err != nil {
		c.Fatalf("Can't listen %s", err)
	}
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	client, err := NewClient(&Endpoint{Host: "localhost", Port: 15985}, "test", "test")
	c.Assert(err, IsNil)
	shell, err := client.CreateShell()
	if err != nil {
		c.Fatalf("Can't create shell %s", err)
	}
	c.Assert(shell.ShellId, Equals, "67A74734-DD32-4F10-89DE-49A060483810")
}
