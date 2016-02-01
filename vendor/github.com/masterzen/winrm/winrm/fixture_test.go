package winrm

import (
	"fmt"
	. "gopkg.in/check.v1"
	"strings"
)

var (
	createShellResponse = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:x="http://schemas.xmlsoap.org/ws/2004/09/transfer" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd">
	<s:Header>
	    <a:Action>http://schemas.xmlsoap.org/ws/2004/09/transfer/CreateResponse</a:Action>
	    <a:MessageID>uuid:195078CF-804B-41F7-A246-9CB3C1A41A9A</a:MessageID>
	    <a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To>
	    <a:RelatesTo>uuid:D00059E8-57D6-4035-AD8D-3EDC495DA163</a:RelatesTo>
	</s:Header>
	<s:Body>
	    <x:ResourceCreated>
	        <a:Address>http://107.20.128.235:5985/wsman</a:Address>
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

	executeCommandResponse = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:x="http://schemas.xmlsoap.org/ws/2004/09/transfer" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandResponse</a:Action><a:MessageID>uuid:D9E108AA-E32B-45E3-8601-E9C70999D3BA</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:F530804C-6D02-4FA9-AE78-1997750594BA</a:RelatesTo></s:Header><s:Body><rsp:CommandResponse><rsp:CommandId>1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4</rsp:CommandId></rsp:CommandResponse></s:Body></s:Envelope>`

	outputResponse = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse</a:Action><a:MessageID>uuid:AAD46BD4-6315-4C3C-93D4-94A55773287D</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:18A52A06-9027-41DC-8850-3F244595AF62</a:RelatesTo></s:Header><s:Body><rsp:ReceiveResponse><rsp:Stream Name="stdout" CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4">VGhhdCdzIGFsbCBmb2xrcyEhIQ==</rsp:Stream><rsp:Stream Name="stderr" CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4">VGhpcyBpcyBzdGRlcnIsIEknbSBwcmV0dHkgc3VyZSE=</rsp:Stream><rsp:CommandState CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Running"></rsp:CommandState></rsp:ReceiveResponse></s:Body></s:Envelope>`

	singleOutputResponse = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse</a:Action><a:MessageID>uuid:AAD46BD4-6315-4C3C-93D4-94A55773287D</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:18A52A06-9027-41DC-8850-3F244595AF62</a:RelatesTo></s:Header><s:Body><rsp:ReceiveResponse><rsp:Stream Name="stdout" CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4">VGhhdCdzIGFsbCBmb2xrcyEhIQ==</rsp:Stream><rsp:CommandState CommandId="1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Running"></rsp:CommandState></rsp:ReceiveResponse></s:Body></s:Envelope>`

	doneCommandResponse = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse</a:Action><a:MessageID>uuid:206F8145-683D-4987-949B-E099F999F088</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:6c68191c-8385-4816-506a-0769cb9f3f4e</a:RelatesTo></s:Header><s:Body><rsp:ReceiveResponse><rsp:CommandState CommandId="4531DAA3-60C2-4CAD-9FCA-F433101DAC8A" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done"><rsp:ExitCode>123</rsp:ExitCode></rsp:CommandState></rsp:ReceiveResponse></s:Body></s:Envelope>
	`
	doneCommandExitCode0Response = `<s:Envelope xml:lang="en-US" xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"><s:Header><a:Action>http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse</a:Action><a:MessageID>uuid:206F8145-683D-4987-949B-E099F999F088</a:MessageID><a:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:To><a:RelatesTo>uuid:6c68191c-8385-4816-506a-0769cb9f3f4e</a:RelatesTo></s:Header><s:Body><rsp:ReceiveResponse><rsp:CommandState CommandId="4531DAA3-60C2-4CAD-9FCA-F433101DAC8A" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done"><rsp:ExitCode>0</rsp:ExitCode></rsp:CommandState></rsp:ReceiveResponse></s:Body></s:Envelope>`
)

type containsChecker struct {
	*CheckerInfo
}

// The Contains checker verifies that the obtained value contains
// the expected value, according to usual Go semantics for strings.Contains.
//
// For example:
//
//     c.Assert(haystack, Contains, "needle")
//
var Contains Checker = &containsChecker{
	&CheckerInfo{Name: "contains", Params: []string{"obtained", "expected"}},
}

func (checker *containsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return matches(params[0], params[1])
}

func matches(haystack, needle interface{}) (result bool, error string) {
	neStr, ok := needle.(string)
	if !ok {
		return false, "Expected value must be a string"
	}
	valueStr, valueIsStr := haystack.(string)
	if !valueIsStr {
		if valueWithStr, valueHasStr := haystack.(fmt.Stringer); valueHasStr {
			valueStr, valueIsStr = valueWithStr.String(), true
		}
	}
	if valueIsStr {
		return strings.Contains(valueStr, neStr), ""
	}
	return false, "Obtained value is not a string and has no .String()"
}
