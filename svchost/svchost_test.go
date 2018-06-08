package svchost

import "testing"

func TestForDisplay(t *testing.T) {
	tests := []struct {
		Input string
		Want  string
	}{
		{
			"",
			"",
		},
		{
			"example.com",
			"example.com",
		},
		{
			"invalid",
			"invalid",
		},
		{
			"localhost",
			"localhost",
		},
		{
			"localhost:1211",
			"localhost:1211",
		},
		{
			"HashiCorp.com",
			"hashicorp.com",
		},
		{
			"Испытание.com",
			"испытание.com",
		},
		{
			"münchen.de", // this is a precomposed u with diaeresis
			"münchen.de", // this is a precomposed u with diaeresis
		},
		{
			"münchen.de", // this is a separate u and combining diaeresis
			"münchen.de",  // this is a precomposed u with diaeresis
		},
		{
			"example.com:443",
			"example.com",
		},
		{
			"example.com:81",
			"example.com:81",
		},
		{
			"example.com:boo",
			"example.com:boo", // invalid, but tolerated for display purposes
		},
		{
			"example.com:boo:boo",
			"example.com:boo:boo", // invalid, but tolerated for display purposes
		},
		{
			"example.com:081",
			"example.com:81",
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got := ForDisplay(test.Input)
			if got != test.Want {
				t.Errorf("wrong result\ninput: %s\ngot:   %s\nwant:  %s", test.Input, got, test.Want)
			}
		})
	}
}

func TestForComparison(t *testing.T) {
	tests := []struct {
		Input string
		Want  string
		Err   bool
	}{
		{
			"",
			"",
			true,
		},
		{
			"example.com",
			"example.com",
			false,
		},
		{
			"example.com:443",
			"example.com",
			false,
		},
		{
			"example.com:81",
			"example.com:81",
			false,
		},
		{
			"example.com:081",
			"example.com:81",
			false,
		},
		{
			"invalid",
			"invalid",
			false, // the "invalid" TLD is, confusingly, a valid hostname syntactically
		},
		{
			"localhost", // supported for local testing only
			"localhost",
			false,
		},
		{
			"localhost:1211", // supported for local testing only
			"localhost:1211",
			false,
		},
		{
			"HashiCorp.com",
			"hashicorp.com",
			false,
		},
		{
			"Испытание.com",
			"xn--80akhbyknj4f.com",
			false,
		},
		{
			"münchen.de", // this is a precomposed u with diaeresis
			"xn--mnchen-3ya.de",
			false,
		},
		{
			"münchen.de", // this is a separate u and combining diaeresis
			"xn--mnchen-3ya.de",
			false,
		},
		{
			"blah..blah",
			"",
			true,
		},
		{
			"example.com:boo",
			"",
			true,
		},
		{
			"example.com:80:boo",
			"",
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got, err := ForComparison(test.Input)
			if (err != nil) != test.Err {
				if test.Err {
					t.Error("unexpected success; want error")
				} else {
					t.Errorf("unexpected error; want success\nerror: %s", err)
				}
			}
			if string(got) != test.Want {
				t.Errorf("wrong result\ninput: %s\ngot:   %s\nwant:  %s", test.Input, got, test.Want)
			}
		})
	}
}

func TestHostnameForDisplay(t *testing.T) {
	tests := []struct {
		Input string
		Want  string
	}{
		{
			"example.com",
			"example.com",
		},
		{
			"example.com:81",
			"example.com:81",
		},
		{
			"xn--80akhbyknj4f.com",
			"испытание.com",
		},
		{
			"xn--80akhbyknj4f.com:8080",
			"испытание.com:8080",
		},
		{
			"xn--mnchen-3ya.de",
			"münchen.de", // this is a precomposed u with diaeresis
		},
	}

	for _, test := range tests {
		t.Run(test.Input, func(t *testing.T) {
			got := Hostname(test.Input).ForDisplay()
			if got != test.Want {
				t.Errorf("wrong result\ninput: %s\ngot:   %s\nwant:  %s", test.Input, got, test.Want)
			}
		})
	}
}
