package api

import (
	"testing"

	"github.com/scaleway/scaleway-cli/pkg/scwversion"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewScalewayAPI(t *testing.T) {
	Convey("Testing NewScalewayAPI()", t, func() {
		api, err := NewScalewayAPI("my-organization", "my-token", scwversion.UserAgent(), "")
		So(err, ShouldBeNil)
		So(api, ShouldNotBeNil)
		So(api.Token, ShouldEqual, "my-token")
		So(api.Organization, ShouldEqual, "my-organization")
		So(api.Cache, ShouldNotBeNil)
		So(api.client, ShouldNotBeNil)
		So(api.Logger, ShouldNotBeNil)
	})
}
