// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016-2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin_test

import (
	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/builtin"
	"github.com/snapcore/snapd/interfaces/dbus"
	"github.com/snapcore/snapd/interfaces/seccomp"
	"github.com/snapcore/snapd/interfaces/udev"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/testutil"
)

type UDisks2InterfaceSuite struct {
	iface    interfaces.Interface
	slotInfo *snap.SlotInfo
	slot     *interfaces.ConnectedSlot
	plugInfo *snap.PlugInfo
	plug     *interfaces.ConnectedPlug
}

var _ = Suite(&UDisks2InterfaceSuite{
	iface: builtin.MustInterface("udisks2"),
})

const udisks2ConsumerYaml = `name: consumer
apps:
 app:
  plugs: [udisks2]
`

const udisks2ConsumerTwoAppsYaml = `name: consumer
apps:
 app1:
  plugs: [udisks2]
 app2:
  plugs: [udisks2]
`

const udisks2ConsumerThreeAppsYaml = `name: consumer
apps:
 app1:
  plugs: [udisks2]
 app2:
  plugs: [udisks2]
 app3:
`

const udisks2ProducerYaml = `name: producer
apps:
 app:
  slots: [udisks2]
`

const udisks2ProducerTwoAppsYaml = `name: producer
apps:
 app1:
  slots: [udisks2]
 app2:
  slots: [udisks2]
`

const udisks2ProducerThreeAppsYaml = `name: producer
apps:
 app1:
  slots: [udisks2]
 app2:
 app3:
  slots: [udisks2]
`

func (s *UDisks2InterfaceSuite) SetUpTest(c *C) {
	s.plug, s.plugInfo = MockConnectedPlug(c, udisks2ConsumerYaml, nil, "udisks2")
	s.slot, s.slotInfo = MockConnectedSlot(c, udisks2ProducerYaml, nil, "udisks2")
}

func (s *UDisks2InterfaceSuite) TestName(c *C) {
	c.Assert(s.iface.Name(), Equals, "udisks2")
}

func (s *UDisks2InterfaceSuite) TestSanitizeSlot(c *C) {
	c.Assert(interfaces.SanitizeSlot(s.iface, s.slotInfo), IsNil)
}

func (s *UDisks2InterfaceSuite) TestAppArmorSpec(c *C) {
	// The label uses short form when exactly one app is bound to the udisks2 slot
	spec := &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `peer=(label="snap.producer.app"),`)

	// The label glob when all apps are bound to the udisks2 slot
	slot, _ := MockConnectedSlot(c, udisks2ProducerTwoAppsYaml, nil, "udisks2")
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `peer=(label="snap.producer.*"),`)

	// The label uses alternation when some, but not all, apps is bound to the udisks2 slot
	slot, _ = MockConnectedSlot(c, udisks2ProducerThreeAppsYaml, nil, "udisks2")
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `peer=(label="snap.producer.{app1,app3}"),`)

	// The label uses short form when exactly one app is bound to the udisks2 plug
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedSlot(s.iface, s.plug, s.slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `peer=(label="snap.consumer.app"),`)

	// The label glob when all apps are bound to the udisks2 plug
	plug, _ := MockConnectedPlug(c, udisks2ConsumerTwoAppsYaml, nil, "udisks2")
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedSlot(s.iface, plug, s.slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `peer=(label="snap.consumer.*"),`)

	// The label uses alternation when some, but not all, apps is bound to the udisks2 plug
	plug, _ = MockConnectedPlug(c, udisks2ConsumerThreeAppsYaml, nil, "udisks2")
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedSlot(s.iface, plug, s.slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `peer=(label="snap.consumer.{app1,app2}"),`)

	// permanent slot have a non-nil security snippet for apparmor
	spec = &apparmor.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.slot), IsNil)
	c.Assert(spec.AddPermanentSlot(s.iface, s.slotInfo), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app", "snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `peer=(label="snap.producer.app"),`)
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `peer=(label=unconfined),`)
}

func (s *UDisks2InterfaceSuite) TestDBusSpec(c *C) {
	spec := &dbus.Specification{}
	c.Assert(spec.AddConnectedPlug(s.iface, s.plug, s.slot), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.consumer.app"})
	c.Assert(spec.SnippetForTag("snap.consumer.app"), testutil.Contains, `<policy context="default">`)

	spec = &dbus.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.slotInfo), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, `<policy user="root">`)
}

func (s *UDisks2InterfaceSuite) TestUDevSpec(c *C) {
	spec := &udev.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.slotInfo), IsNil)
	c.Assert(spec.Snippets(), HasLen, 3)
	c.Assert(spec.Snippets()[0], testutil.Contains, `LABEL="udisks_probe_end"`)
	c.Assert(spec.Snippets(), testutil.Contains, `# udisks2
SUBSYSTEM=="block", TAG+="snap_producer_app"`)
	c.Assert(spec.Snippets(), testutil.Contains, `# udisks2
SUBSYSTEM=="usb", TAG+="snap_producer_app"`)
}

func (s *UDisks2InterfaceSuite) TestSecCompSpec(c *C) {
	spec := &seccomp.Specification{}
	c.Assert(spec.AddPermanentSlot(s.iface, s.slotInfo), IsNil)
	c.Assert(spec.SecurityTags(), DeepEquals, []string{"snap.producer.app"})
	c.Assert(spec.SnippetForTag("snap.producer.app"), testutil.Contains, "mount\n")
}

func (s *UDisks2InterfaceSuite) TestStaticInfo(c *C) {
	si := interfaces.StaticInfoOf(s.iface)
	c.Assert(si.ImplicitOnCore, Equals, false)
	c.Assert(si.ImplicitOnClassic, Equals, false)
	c.Assert(si.Summary, Equals, `allows operating as or interacting with the UDisks2 service`)
	c.Assert(si.BaseDeclarationSlots, testutil.Contains, "udisks2")
}

func (s *UDisks2InterfaceSuite) TestAutoConnect(c *C) {
	// FIXME: fix AutoConnect methods
	c.Assert(s.iface.AutoConnect(&interfaces.Plug{PlugInfo: s.plugInfo}, &interfaces.Slot{SlotInfo: s.slotInfo}), Equals, true)
}

func (s *UDisks2InterfaceSuite) TestInterfaces(c *C) {
	c.Assert(builtin.Interfaces(), testutil.DeepContains, s.iface)
}
