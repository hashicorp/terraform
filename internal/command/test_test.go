package command

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/terminal"
)

// These are the main tests for the "terraform test" command.
func TestTest(t *testing.T) {
	t.Run("passes", func(t *testing.T) {
		td := tempDir(t)
		testCopyDir(t, testFixturePath("test-passes"), td)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		streams, close := terminal.StreamsForTesting(t)
		cmd := &TestCommand{
			Meta: Meta{
				Streams: streams,
				View:    views.NewView(streams),
			},
		}
		exitStatus := cmd.Run([]string{"-junit-xml=junit.xml", "-no-color"})
		outp := close(t)
		if got, want := exitStatus, 0; got != want {
			t.Fatalf("wrong exit status %d; want %d\nstderr:\n%s", got, want, outp.Stderr())
		}

		gotStdout := strings.TrimSpace(outp.Stdout())
		wantStdout := strings.TrimSpace(`
Warning: The "terraform test" command is experimental

We'd like to invite adventurous module authors to write integration tests for
their modules using this command, but all of the behaviors of this command
are currently experimental and may change based on feedback.

For more information on the testing experiment, including ongoing research
goals and avenues for feedback, see:
    https://www.terraform.io/docs/language/modules/testing-experiment.html
`)
		if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
			t.Errorf("wrong stdout\n%s", diff)
		}

		gotStderr := strings.TrimSpace(outp.Stderr())
		wantStderr := strings.TrimSpace(`
Success! All of the test assertions passed.
`)
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong stderr\n%s", diff)
		}

		gotXMLSrc, err := ioutil.ReadFile("junit.xml")
		if err != nil {
			t.Fatal(err)
		}
		gotXML := string(bytes.TrimSpace(gotXMLSrc))
		wantXML := strings.TrimSpace(`
<testsuites>
  <errors>0</errors>
  <failures>0</failures>
  <tests>1</tests>
  <testsuite>
    <name>hello</name>
    <tests>1</tests>
    <skipped>0</skipped>
    <errors>0</errors>
    <failures>0</failures>
    <testcase>
      <name>output</name>
      <classname>foo</classname>
    </testcase>
  </testsuite>
</testsuites>
`)
		if diff := cmp.Diff(wantXML, gotXML); diff != "" {
			t.Errorf("wrong JUnit XML\n%s", diff)
		}
	})
	t.Run("fails", func(t *testing.T) {
		td := tempDir(t)
		testCopyDir(t, testFixturePath("test-fails"), td)
		defer os.RemoveAll(td)
		defer testChdir(t, td)()

		streams, close := terminal.StreamsForTesting(t)
		cmd := &TestCommand{
			Meta: Meta{
				Streams: streams,
				View:    views.NewView(streams),
			},
		}
		exitStatus := cmd.Run([]string{"-junit-xml=junit.xml", "-no-color"})
		outp := close(t)
		if got, want := exitStatus, 1; got != want {
			t.Fatalf("wrong exit status %d; want %d\nstderr:\n%s", got, want, outp.Stderr())
		}

		gotStdout := strings.TrimSpace(outp.Stdout())
		wantStdout := strings.TrimSpace(`
Warning: The "terraform test" command is experimental

We'd like to invite adventurous module authors to write integration tests for
their modules using this command, but all of the behaviors of this command
are currently experimental and may change based on feedback.

For more information on the testing experiment, including ongoing research
goals and avenues for feedback, see:
    https://www.terraform.io/docs/language/modules/testing-experiment.html
`)
		if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
			t.Errorf("wrong stdout\n%s", diff)
		}

		gotStderr := strings.TrimSpace(outp.Stderr())
		wantStderr := strings.TrimSpace(`
─── Failed: hello.foo.output (output "foo" value) ───────────────────────────
wrong value
    got:  "foo value boop"
    want: "foo not boop"

─────────────────────────────────────────────────────────────────────────────
`)
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Errorf("wrong stderr\n%s", diff)
		}

		gotXMLSrc, err := ioutil.ReadFile("junit.xml")
		if err != nil {
			t.Fatal(err)
		}
		gotXML := string(bytes.TrimSpace(gotXMLSrc))
		wantXML := strings.TrimSpace(`
<testsuites>
  <errors>0</errors>
  <failures>1</failures>
  <tests>1</tests>
  <testsuite>
    <name>hello</name>
    <tests>1</tests>
    <skipped>0</skipped>
    <errors>0</errors>
    <failures>1</failures>
    <testcase>
      <name>output</name>
      <classname>foo</classname>
      <failure>
        <message>wrong value&#xA;    got:  &#34;foo value boop&#34;&#xA;    want: &#34;foo not boop&#34;&#xA;</message>
      </failure>
    </testcase>
  </testsuite>
</testsuites>
`)
		if diff := cmp.Diff(wantXML, gotXML); diff != "" {
			t.Errorf("wrong JUnit XML\n%s", diff)
		}
	})

}
