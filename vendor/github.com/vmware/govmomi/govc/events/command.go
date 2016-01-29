/*
Copyright (c) 2015 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/govc/cli"
	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

type events struct {
	*flags.DatacenterFlag

	Max int
}

func init() {
	cli.Register("events", &events{})
}

func (cmd *events) Register(ctx context.Context, f *flag.FlagSet) {
	cmd.DatacenterFlag, ctx = flags.NewDatacenterFlag(ctx)
	cmd.DatacenterFlag.Register(ctx, f)

	f.IntVar(&cmd.Max, "n", 25, "Output the last N events")
}

func (cmd *events) Process(ctx context.Context) error {
	if err := cmd.DatacenterFlag.Process(ctx); err != nil {
		return err
	}
	return nil
}

func (cmd *events) Usage() string {
	return "[PATH]..."
}

func (cmd *events) Run(ctx context.Context, f *flag.FlagSet) error {
	c, err := cmd.Client()
	if err != nil {
		return err
	}

	m := event.NewManager(c)

	objs, err := cmd.ManagedObjects(ctx, f.Args())
	if err != nil {
		return err
	}

	var events []types.BaseEvent

	for _, o := range objs {
		filter := types.EventFilterSpec{
			Entity: &types.EventFilterSpecByEntity{
				Entity:    o,
				Recursion: types.EventFilterSpecRecursionOptionAll,
			},
		}

		collector, err := m.CreateCollectorForEvents(ctx, filter)
		if err != nil {
			return fmt.Errorf("[%#v] %s", o, err)
		}
		defer collector.Destroy(ctx)

		err = collector.SetPageSize(ctx, cmd.Max)
		if err != nil {
			return err
		}

		page, err := collector.LatestPage(ctx)
		if err != nil {
			return err
		}

		events = append(events, page...)
	}

	event.Sort(events)

	for _, e := range events {
		cat, err := m.EventCategory(ctx, e)
		if err != nil {
			return err
		}

		event := e.GetEvent()
		msg := strings.TrimSpace(event.FullFormattedMessage)

		if t, ok := e.(*types.TaskEvent); ok {
			msg = fmt.Sprintf("%s (target=%s %s)", msg, t.Info.Entity.Type, t.Info.EntityName)
		}

		fmt.Fprintf(os.Stdout, "[%s] [%s] %s\n",
			event.CreatedTime.Local().Format(time.ANSIC),
			cat, msg)
	}

	return nil
}
