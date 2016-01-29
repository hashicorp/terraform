package winrm

import (
	. "gopkg.in/check.v1"
)

func (s *WinRMSuite) TestPowershell(c *C) {
	psCmd := Powershell("dir")
	c.Assert(psCmd, Equals, "powershell.exe -EncodedCommand ZABpAHIA")
}
