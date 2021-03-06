// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
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

package main_test

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/check.v1"

	snaprun "github.com/snapcore/snapd/cmd/snap"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/snap"
	"github.com/snapcore/snapd/snap/snaptest"
	"github.com/snapcore/snapd/testutil"
)

var mockYaml = []byte(`name: snapname
version: 1.0
apps:
 app:
  command: run-app
hooks:
 configure:
`)

func (s *SnapSuite) TestInvalidParameters(c *check.C) {
	invalidParameters := []string{"run", "--hook=configure", "--command=command-name", "snap-name"}
	_, err := snaprun.Parser().ParseArgs(invalidParameters)
	c.Check(err, check.ErrorMatches, ".*cannot use --hook and --command together.*")

	invalidParameters = []string{"run", "-r=1", "--command=command-name", "snap-name"}
	_, err = snaprun.Parser().ParseArgs(invalidParameters)
	c.Check(err, check.ErrorMatches, ".*-r can only be used with --hook.*")

	invalidParameters = []string{"run", "-r=1", "snap-name"}
	_, err = snaprun.Parser().ParseArgs(invalidParameters)
	c.Check(err, check.ErrorMatches, ".*-r can only be used with --hook.*")

	invalidParameters = []string{"run", "--hook=configure", "foo", "bar", "snap-name"}
	_, err = snaprun.Parser().ParseArgs(invalidParameters)
	c.Check(err, check.ErrorMatches, ".*too many arguments for hook \"configure\": bar.*")
}

func (s *SnapSuite) TestSnapRunAppIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execArg0 := ""
	execArgs := []string{}
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execArg0 = arg0
		execArgs = args
		execEnv = envv
		return nil
	})
	defer restorer()

	// and run it!
	rest, err := snaprun.Parser().ParseArgs([]string{"run", "snapname.app", "--arg1", "arg2"})
	c.Assert(err, check.IsNil)
	c.Assert(rest, check.DeepEquals, []string{"snapname.app", "--arg1", "arg2"})
	c.Check(execArg0, check.Equals, "/usr/bin/ubuntu-core-launcher")
	c.Check(execArgs, check.DeepEquals, []string{
		"/usr/bin/ubuntu-core-launcher",
		"snap.snapname.app",
		"snap.snapname.app",
		"/usr/lib/snapd/snap-exec",
		"snapname.app",
		"--arg1", "arg2"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=42")
}

func (s *SnapSuite) TestSnapRunAppWithCommandIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execArg0 := ""
	execArgs := []string{}
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execArg0 = arg0
		execArgs = args
		execEnv = envv
		return nil
	})
	defer restorer()

	// and run it!
	err := snaprun.SnapRunApp("snapname.app", "my-command", []string{"arg1", "arg2"})
	c.Assert(err, check.IsNil)
	c.Check(execArg0, check.Equals, "/usr/bin/ubuntu-core-launcher")
	c.Check(execArgs, check.DeepEquals, []string{
		"/usr/bin/ubuntu-core-launcher",
		"snap.snapname.app",
		"snap.snapname.app",
		"/usr/lib/snapd/snap-exec",
		"--command=my-command", "snapname.app",
		"arg1", "arg2"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=42")
}

func (s *SnapSuite) TestSnapRunCreateDataDirs(c *check.C) {
	info, err := snap.InfoFromSnapYaml(mockYaml)
	c.Assert(err, check.IsNil)
	info.SideInfo.Revision = snap.R(42)

	fakeHome := c.MkDir()
	restorer := snaprun.MockUserCurrent(func() (*user.User, error) {
		return &user.User{HomeDir: fakeHome}, nil
	})
	defer restorer()

	err = snaprun.CreateUserDataDirs(info)
	c.Assert(err, check.IsNil)
	c.Check(osutil.FileExists(filepath.Join(fakeHome, "/snap/snapname/42")), check.Equals, true)
	c.Check(osutil.FileExists(filepath.Join(fakeHome, "/snap/snapname/common")), check.Equals, true)
}

func (s *SnapSuite) TestSnapRunHookIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execArg0 := ""
	execArgs := []string{}
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execArg0 = arg0
		execArgs = args
		execEnv = envv
		return nil
	})
	defer restorer()

	// Run a hook from the active revision
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--hook=configure", "snapname"})
	c.Assert(err, check.IsNil)
	c.Check(execArg0, check.Equals, "/usr/bin/ubuntu-core-launcher")
	c.Check(execArgs, check.DeepEquals, []string{
		"/usr/bin/ubuntu-core-launcher",
		"snap.snapname.hook.configure",
		"snap.snapname.hook.configure",
		"/usr/lib/snapd/snap-exec",
		"--hook=configure", "snapname"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=42")
}

func (s *SnapSuite) TestSnapRunHookUnsetRevisionIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execArg0 := ""
	execArgs := []string{}
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execArg0 = arg0
		execArgs = args
		execEnv = envv
		return nil
	})
	defer restorer()

	// Specifically pass "unset" which would use the active version.
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--hook=configure", "-r=unset", "snapname"})
	c.Assert(err, check.IsNil)
	c.Check(execArg0, check.Equals, "/usr/bin/ubuntu-core-launcher")
	c.Check(execArgs, check.DeepEquals, []string{
		"/usr/bin/ubuntu-core-launcher",
		"snap.snapname.hook.configure",
		"snap.snapname.hook.configure",
		"/usr/lib/snapd/snap-exec",
		"--hook=configure", "snapname"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=42")
}

func (s *SnapSuite) TestSnapRunHookSpecificRevisionIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	// Create both revisions 41 and 42
	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(41),
	})
	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execArg0 := ""
	execArgs := []string{}
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execArg0 = arg0
		execArgs = args
		execEnv = envv
		return nil
	})
	defer restorer()

	// Run a hook on revision 41
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--hook=configure", "-r=41", "snapname"})
	c.Assert(err, check.IsNil)
	c.Check(execArg0, check.Equals, "/usr/bin/ubuntu-core-launcher")
	c.Check(execArgs, check.DeepEquals, []string{
		"/usr/bin/ubuntu-core-launcher",
		"snap.snapname.hook.configure",
		"snap.snapname.hook.configure",
		"/usr/lib/snapd/snap-exec",
		"--hook=configure", "snapname"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=41")
}

func (s *SnapSuite) TestSnapRunHookMissingRevisionIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	// Only create revision 42
	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		return nil
	})
	defer restorer()

	// Attempt to run a hook on revision 41, which doesn't exist
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--hook=configure", "-r=41", "snapname"})
	c.Assert(err, check.NotNil)
	c.Check(err, check.ErrorMatches, "cannot find .*")
}

func (s *SnapSuite) TestSnapRunHookInvalidRevisionIntegration(c *check.C) {
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--hook=configure", "-r=invalid", "snapname"})
	c.Assert(err, check.NotNil)
	c.Check(err, check.ErrorMatches, "invalid snap revision: \"invalid\"")
}

func (s *SnapSuite) TestSnapRunHookMissingHookIntegration(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	// Only create revision 42
	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	called := false
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		called = true
		return nil
	})
	defer restorer()

	err := snaprun.SnapRunHook("snapname", "unset", "missing-hook")
	c.Assert(err, check.IsNil)
	c.Check(called, check.Equals, false)
}

func (s *SnapSuite) mockServer(c *check.C) {
	n := 0
	s.RedirectClientToTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch n {
		case 0:
			c.Check(r.Method, check.Equals, "GET")
			c.Check(r.URL.Path, check.Equals, "/v2/snaps")
			fmt.Fprintln(w, `{"type": "sync", "result": [{"name": "snapname", "status": "active", "version": "1.0", "developer": "someone", "revision":42}]}`)
		default:
			c.Fatalf("expected to get 1 requests, now on %d", n+1)
		}

		n++
	})
}

func (s *SnapSuite) TestSnapRunErorsForUnknownRunArg(c *check.C) {
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--unknown", "snapname.app", "--arg1", "arg2"})
	c.Assert(err, check.ErrorMatches, "unknown flag `unknown'")
}

func (s *SnapSuite) TestSnapRunErorsForMissingApp(c *check.C) {
	_, err := snaprun.Parser().ParseArgs([]string{"run", "--command=shell"})
	c.Assert(err, check.ErrorMatches, "need the application to run as argument")
}

func (s *SnapSuite) TestSnapRunSaneEnvironmentHandling(c *check.C) {
	// mock installed snap
	dirs.SetRootDir(c.MkDir())
	defer func() { dirs.SetRootDir("/") }()

	snaptest.MockSnap(c, string(mockYaml), &snap.SideInfo{
		Revision: snap.R(42),
	})

	// and mock the server
	s.mockServer(c)

	// redirect exec
	execEnv := []string{}
	restorer := snaprun.MockSyscallExec(func(arg0 string, args []string, envv []string) error {
		execEnv = envv
		return nil
	})
	defer restorer()

	// set a SNAP{,_*} variable in the environment
	os.Setenv("SNAP_NAME", "something-else")
	os.Setenv("SNAP_ARCH", "PDP-7")
	defer os.Unsetenv("SNAP_NAME")
	defer os.Unsetenv("SNAP_ARCH")
	// but unreleated stuff is ok
	os.Setenv("SNAP_THE_WORLD", "YES")
	defer os.Unsetenv("SNAP_THE_WORLD")

	// and ensure those SNAP_ vars get overriden
	rest, err := snaprun.Parser().ParseArgs([]string{"run", "snapname.app", "--arg1", "arg2"})
	c.Assert(err, check.IsNil)
	c.Assert(rest, check.DeepEquals, []string{"snapname.app", "--arg1", "arg2"})
	c.Check(execEnv, testutil.Contains, "SNAP_REVISION=42")
	c.Check(execEnv, check.Not(testutil.Contains), "SNAP_NAME=something-else")
	c.Check(execEnv, check.Not(testutil.Contains), "SNAP_ARCH=PDP-7")
	c.Check(execEnv, testutil.Contains, "SNAP_THE_WORLD=YES")
}
