package cidr

import (
	"fmt"
	"net"
	"testing"
)

func TestSubnet(t *testing.T) {
	type Case struct {
		Base   string
		Bits   int
		Num    int
		Output string
		Error  bool
	}

	cases := []Case{
		Case{
			Base:   "192.168.2.0/20",
			Bits:   4,
			Num:    6,
			Output: "192.168.6.0/24",
		},
		Case{
			Base:   "192.168.2.0/20",
			Bits:   4,
			Num:    0,
			Output: "192.168.0.0/24",
		},
		Case{
			Base:   "192.168.0.0/31",
			Bits:   1,
			Num:    1,
			Output: "192.168.0.1/32",
		},
		Case{
			Base:   "192.168.0.0/21",
			Bits:   4,
			Num:    7,
			Output: "192.168.3.128/25",
		},
		Case{
			Base:   "fe80::/48",
			Bits:   16,
			Num:    6,
			Output: "fe80:0:0:6::/64",
		},
		Case{
			Base:   "fe80::/49",
			Bits:   16,
			Num:    7,
			Output: "fe80:0:0:3:8000::/65",
		},
		Case{
			Base:  "192.168.2.0/31",
			Bits:  2,
			Num:   0,
			Error: true, // not enough bits to expand into
		},
		Case{
			Base:  "fe80::/126",
			Bits:  4,
			Num:   0,
			Error: true, // not enough bits to expand into
		},
		Case{
			Base:  "192.168.2.0/24",
			Bits:  4,
			Num:   16,
			Error: true, // can't fit 16 into 4 bits
		},
	}

	for _, testCase := range cases {
		_, base, _ := net.ParseCIDR(testCase.Base)
		gotNet, err := Subnet(base, testCase.Bits, testCase.Num)
		desc := fmt.Sprintf("Subnet(%#v,%#v,%#v)", testCase.Base, testCase.Bits, testCase.Num)
		if err != nil {
			if !testCase.Error {
				t.Errorf("%s failed: %s", desc, err.Error())
			}
		} else {
			got := gotNet.String()
			if testCase.Error {
				t.Errorf("%s = %s; want error", desc, got)
			} else {
				if got != testCase.Output {
					t.Errorf("%s = %s; want %s", desc, got, testCase.Output)
				}
			}
		}
	}
}

func TestHost(t *testing.T) {
	type Case struct {
		Range  string
		Num    int
		Output string
		Error  bool
	}

	cases := []Case{
		Case{
			Range:  "192.168.2.0/20",
			Num:    6,
			Output: "192.168.0.6",
		},
		Case{
			Range:  "192.168.0.0/20",
			Num:    257,
			Output: "192.168.1.1",
		},
		Case{
			Range: "192.168.1.0/24",
			Num:   256,
			Error: true, // only 0-255 will fit in 8 bits
		},
	}

	for _, testCase := range cases {
		_, network, _ := net.ParseCIDR(testCase.Range)
		gotIP, err := Host(network, testCase.Num)
		desc := fmt.Sprintf("Host(%#v,%#v)", testCase.Range, testCase.Num)
		if err != nil {
			if !testCase.Error {
				t.Errorf("%s failed: %s", desc, err.Error())
			}
		} else {
			got := gotIP.String()
			if testCase.Error {
				t.Errorf("%s = %s; want error", desc, got)
			} else {
				if got != testCase.Output {
					t.Errorf("%s = %s; want %s", desc, got, testCase.Output)
				}
			}
		}
	}
}

func TestAddressRange(t *testing.T) {
	type Case struct {
		Range string
		First string
		Last  string
	}

	cases := []Case{
		Case{
			Range: "192.168.0.0/16",
			First: "192.168.0.0",
			Last:  "192.168.255.255",
		},
		Case{
			Range: "192.168.0.0/17",
			First: "192.168.0.0",
			Last:  "192.168.127.255",
		},
		Case{
			Range: "fe80::/64",
			First: "fe80::",
			Last:  "fe80::ffff:ffff:ffff:ffff",
		},
	}

	for _, testCase := range cases {
		_, network, _ := net.ParseCIDR(testCase.Range)
		firstIP, lastIP := AddressRange(network)
		desc := fmt.Sprintf("AddressRange(%#v)", testCase.Range)
		gotFirstIP := firstIP.String()
		gotLastIP := lastIP.String()
		if gotFirstIP != testCase.First {
			t.Errorf("%s first is %s; want %s", desc, gotFirstIP, testCase.First)
		}
		if gotLastIP != testCase.Last {
			t.Errorf("%s last is %s; want %s", desc, gotLastIP, testCase.Last)
		}
	}

}

func TestAddressCount(t *testing.T) {
	type Case struct {
		Range string
		Count uint64
	}

	cases := []Case{
		Case{
			Range: "192.168.0.0/16",
			Count: 65536,
		},
		Case{
			Range: "192.168.0.0/17",
			Count: 32768,
		},
		Case{
			Range: "192.168.0.0/32",
			Count: 1,
		},
		Case{
			Range: "192.168.0.0/31",
			Count: 2,
		},
		Case{
			Range: "0.0.0.0/0",
			Count: 4294967296,
		},
		Case{
			Range: "0.0.0.0/1",
			Count: 2147483648,
		},
		Case{
			Range: "::/65",
			Count: 9223372036854775808,
		},
		Case{
			Range: "::/128",
			Count: 1,
		},
		Case{
			Range: "::/127",
			Count: 2,
		},
	}

	for _, testCase := range cases {
		_, network, _ := net.ParseCIDR(testCase.Range)
		gotCount := AddressCount(network)
		desc := fmt.Sprintf("AddressCount(%#v)", testCase.Range)
		if gotCount != testCase.Count {
			t.Errorf("%s = %d; want %d", desc, gotCount, testCase.Count)
		}
	}

}
