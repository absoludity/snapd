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

package auth_test

import (
	"os"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/overlord/auth"
	"github.com/snapcore/snapd/overlord/state"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type authSuite struct {
	state *state.State
}

var _ = Suite(&authSuite{})

func (as *authSuite) SetUpTest(c *C) {
	as.state = state.New(nil)
}

func (as *authSuite) TestNewUser(c *C) {
	as.state.Lock()
	user, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()

	expected := &auth.UserState{
		ID:              1,
		Username:        "username",
		Macaroon:        "macaroon",
		Discharges:      []string{"discharge"},
		StoreMacaroon:   "macaroon",
		StoreDischarges: []string{"discharge"},
	}
	c.Check(err, IsNil)
	c.Check(user, DeepEquals, expected)

	as.state.Lock()
	userFromState, err := auth.User(as.state, 1)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, expected)
}

func (as *authSuite) TestNewUserSortsDischarges(c *C) {
	as.state.Lock()
	user, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge2", "discharge1"})
	as.state.Unlock()

	expected := &auth.UserState{
		ID:              1,
		Username:        "username",
		Macaroon:        "macaroon",
		Discharges:      []string{"discharge1", "discharge2"},
		StoreMacaroon:   "macaroon",
		StoreDischarges: []string{"discharge1", "discharge2"},
	}
	c.Check(err, IsNil)
	c.Check(user, DeepEquals, expected)

	as.state.Lock()
	userFromState, err := auth.User(as.state, 1)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, expected)
}

func (as *authSuite) TestNewUserAddsToExistent(c *C) {
	as.state.Lock()
	firstUser, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	// adding a new one
	as.state.Lock()
	user, err := auth.NewUser(as.state, "new_username", "new_macaroon", []string{"new_discharge"})
	as.state.Unlock()
	expected := &auth.UserState{
		ID:              2,
		Username:        "new_username",
		Macaroon:        "new_macaroon",
		Discharges:      []string{"new_discharge"},
		StoreMacaroon:   "new_macaroon",
		StoreDischarges: []string{"new_discharge"},
	}
	c.Check(err, IsNil)
	c.Check(user, DeepEquals, expected)

	as.state.Lock()
	userFromState, err := auth.User(as.state, 2)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, expected)

	// first user is still in the state
	as.state.Lock()
	userFromState, err = auth.User(as.state, 1)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, firstUser)
}

func (as *authSuite) TestCheckMacaroonNoAuthData(c *C) {
	as.state.Lock()
	user, err := auth.CheckMacaroon(as.state, "macaroon", []string{"discharge"})
	as.state.Unlock()

	c.Check(err, Equals, auth.ErrInvalidAuth)
	c.Check(user, IsNil)
}

func (as *authSuite) TestCheckMacaroonInvalidAuth(c *C) {
	as.state.Lock()
	user, err := auth.CheckMacaroon(as.state, "other-macaroon", []string{"discharge"})
	as.state.Unlock()

	c.Check(err, Equals, auth.ErrInvalidAuth)
	c.Check(user, IsNil)

	as.state.Lock()
	_, err = auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	user, err = auth.CheckMacaroon(as.state, "other-macaroon", []string{"discharge"})
	as.state.Unlock()

	c.Check(err, Equals, auth.ErrInvalidAuth)
	c.Check(user, IsNil)
}

func (as *authSuite) TestCheckMacaroonValidUser(c *C) {
	as.state.Lock()
	expectedUser, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	user, err := auth.CheckMacaroon(as.state, "macaroon", []string{"discharge"})
	as.state.Unlock()

	c.Check(err, IsNil)
	c.Check(user, DeepEquals, expectedUser)
}

func (as *authSuite) TestUserForNoAuthInState(c *C) {
	as.state.Lock()
	userFromState, err := auth.User(as.state, 42)
	as.state.Unlock()
	c.Check(err, NotNil)
	c.Check(userFromState, IsNil)
}

func (as *authSuite) TestUserForNonExistent(c *C) {
	as.state.Lock()
	_, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	userFromState, err := auth.User(as.state, 42)
	c.Check(err, ErrorMatches, "invalid user")
	c.Check(userFromState, IsNil)
}

func (as *authSuite) TestUser(c *C) {
	as.state.Lock()
	user, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	userFromState, err := auth.User(as.state, 1)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, user)
}

func (as *authSuite) TestUpdateUser(c *C) {
	as.state.Lock()
	user, _ := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()

	user.Username = "different"
	user.StoreDischarges = []string{"updated-discharge"}

	as.state.Lock()
	err := auth.UpdateUser(as.state, user)
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	userFromState, err := auth.User(as.state, user.ID)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, user)
}

func (as *authSuite) TestUpdateUserInvalid(c *C) {
	as.state.Lock()
	_, _ = auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()

	user := &auth.UserState{
		ID:       102,
		Username: "username",
		Macaroon: "macaroon",
	}

	as.state.Lock()
	err := auth.UpdateUser(as.state, user)
	as.state.Unlock()
	c.Assert(err, ErrorMatches, "invalid user")
}

func (as *authSuite) TestRemove(c *C) {
	as.state.Lock()
	user, err := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	_, err = auth.User(as.state, user.ID)
	as.state.Unlock()
	c.Check(err, IsNil)

	as.state.Lock()
	err = auth.RemoveUser(as.state, user.ID)
	as.state.Unlock()
	c.Assert(err, IsNil)

	as.state.Lock()
	_, err = auth.User(as.state, user.ID)
	as.state.Unlock()
	c.Check(err, ErrorMatches, "invalid user")

	as.state.Lock()
	err = auth.RemoveUser(as.state, user.ID)
	as.state.Unlock()
	c.Assert(err, ErrorMatches, "invalid user")
}

func (as *authSuite) TestSetDevice(c *C) {
	as.state.Lock()
	device, err := auth.Device(as.state)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(device, DeepEquals, &auth.DeviceState{})

	as.state.Lock()
	err = auth.SetDevice(as.state, &auth.DeviceState{Brand: "some-brand"})
	c.Check(err, IsNil)
	device, err = auth.Device(as.state)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(device, DeepEquals, &auth.DeviceState{Brand: "some-brand"})
}

func (as *authSuite) TestAuthContextUpdateUserAuth(c *C) {
	as.state.Lock()
	user, _ := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()

	newDischarges := []string{"updated-discharge"}

	authContext := auth.NewAuthContext(as.state, nil)
	user, err := authContext.UpdateUserAuth(user, newDischarges)
	c.Check(err, IsNil)

	as.state.Lock()
	userFromState, err := auth.User(as.state, user.ID)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, user)
	c.Check(userFromState.Discharges, DeepEquals, []string{"discharge"})
	c.Check(user.StoreDischarges, DeepEquals, newDischarges)
}

func (as *authSuite) TestAuthContextUpdateUserAuthOtherUpdate(c *C) {
	as.state.Lock()
	user, _ := auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	otherUpdateUser := *user
	otherUpdateUser.Macaroon = "macaroon2"
	otherUpdateUser.StoreDischarges = []string{"other-discharges"}
	err := auth.UpdateUser(as.state, &otherUpdateUser)
	as.state.Unlock()
	c.Assert(err, IsNil)

	newDischarges := []string{"updated-discharge"}

	authContext := auth.NewAuthContext(as.state, nil)
	// last discharges win
	curUser, err := authContext.UpdateUserAuth(user, newDischarges)
	c.Assert(err, IsNil)

	as.state.Lock()
	userFromState, err := auth.User(as.state, user.ID)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(userFromState, DeepEquals, curUser)
	c.Check(curUser, DeepEquals, &auth.UserState{
		ID:              user.ID,
		Username:        "username",
		Macaroon:        "macaroon2",
		Discharges:      []string{"discharge"},
		StoreMacaroon:   "macaroon",
		StoreDischarges: newDischarges,
	})
}

func (as *authSuite) TestAuthContextUpdateUserAuthInvalid(c *C) {
	as.state.Lock()
	_, _ = auth.NewUser(as.state, "username", "macaroon", []string{"discharge"})
	as.state.Unlock()

	user := &auth.UserState{
		ID:       102,
		Username: "username",
		Macaroon: "macaroon",
	}

	authContext := auth.NewAuthContext(as.state, nil)
	_, err := authContext.UpdateUserAuth(user, nil)
	c.Assert(err, ErrorMatches, "invalid user")
}

func (as *authSuite) TestAuthContextDeviceForNonExistent(c *C) {
	authContext := auth.NewAuthContext(as.state, nil)

	device, err := authContext.Device()
	c.Check(err, IsNil)
	c.Check(device, DeepEquals, &auth.DeviceState{})
}

func (as *authSuite) TestAuthContextDevice(c *C) {
	device := &auth.DeviceState{Brand: "some-brand"}
	as.state.Lock()
	err := auth.SetDevice(as.state, device)
	as.state.Unlock()
	c.Check(err, IsNil)

	authContext := auth.NewAuthContext(as.state, nil)

	deviceFromState, err := authContext.Device()
	c.Check(err, IsNil)
	c.Check(deviceFromState, DeepEquals, device)
}

func (as *authSuite) TestAuthContextUpdateDeviceAuth(c *C) {
	as.state.Lock()
	device, err := auth.Device(as.state)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(device, DeepEquals, &auth.DeviceState{})

	sessionMacaroon := "the-device-macaroon"

	authContext := auth.NewAuthContext(as.state, nil)
	device, err = authContext.UpdateDeviceAuth(device, sessionMacaroon)
	c.Check(err, IsNil)

	deviceFromState, err := authContext.Device()
	c.Check(err, IsNil)
	c.Check(deviceFromState, DeepEquals, device)
	c.Check(deviceFromState.SessionMacaroon, DeepEquals, sessionMacaroon)
}

func (as *authSuite) TestAuthContextUpdateDeviceAuthOtherUpdate(c *C) {
	as.state.Lock()
	device, _ := auth.Device(as.state)
	otherUpdateDevice := *device
	otherUpdateDevice.SessionMacaroon = "othe-session-macaroon"
	otherUpdateDevice.KeyID = "KEYID"
	err := auth.SetDevice(as.state, &otherUpdateDevice)
	as.state.Unlock()
	c.Check(err, IsNil)

	sessionMacaroon := "the-device-macaroon"

	authContext := auth.NewAuthContext(as.state, nil)
	curDevice, err := authContext.UpdateDeviceAuth(device, sessionMacaroon)
	c.Assert(err, IsNil)

	as.state.Lock()
	deviceFromState, err := auth.Device(as.state)
	as.state.Unlock()
	c.Check(err, IsNil)
	c.Check(deviceFromState, DeepEquals, curDevice)
	c.Check(curDevice, DeepEquals, &auth.DeviceState{
		KeyID:           "KEYID",
		SessionMacaroon: sessionMacaroon,
	})
}

func (as *authSuite) TestAuthContextStoreIDFallback(c *C) {
	authContext := auth.NewAuthContext(as.state, nil)

	storeID, err := authContext.StoreID("store-id")
	c.Assert(err, IsNil)
	c.Check(storeID, Equals, "store-id")
}

func (as *authSuite) TestAuthContextStoreIDFromEnv(c *C) {
	authContext := auth.NewAuthContext(as.state, nil)

	os.Setenv("UBUNTU_STORE_ID", "env-store-id")
	defer os.Unsetenv("UBUNTU_STORE_ID")
	storeID, err := authContext.StoreID("")
	c.Assert(err, IsNil)
	c.Check(storeID, Equals, "env-store-id")
}
func (as *authSuite) TestAuthContextDeviceSessionRequestNilDeviceAssertions(c *C) {
	authContext := auth.NewAuthContext(as.state, nil)

	_, _, err := authContext.DeviceSessionRequest("NONCE")
	c.Check(err, Equals, auth.ErrNoSerial)
}

const (
	exModel = `type: model
authority-id: my-brand
series: 16
brand-id: my-brand
model: baz-3000
architecture: armhf
gadget: gadget
kernel: kernel
store: my-brand-store-id
timestamp: 2016-08-20T13:00:00Z
sign-key-sha3-384: Jv8_JiHiIzJVcO9M55pPdqSDWUvuhfDIBJUS-3VW7F_idjix7Ffn5qMxB21ZQuij

AXNpZw=`

	exSerial = `type: serial
authority-id: my-brand
brand-id: my-brand
model: baz-3000
serial: 9999
device-key:
    AcbBTQRWhcGAARAAtJGIguK7FhSyRxL/6jvdy0zAgGCjC1xVNFzeF76p5G8BXNEEHZUHK+z8Gr2J
    inVrpvhJhllf5Ob2dIMH2YQbC9jE1kjbzvuauQGDqk6tNQm0i3KDeHCSPgVN+PFXPwKIiLrh66Po
    AC7OfR1rFUgCqu0jch0H6Nue0ynvEPiY4dPeXq7mCdpDr5QIAM41L+3hg0OdzvO8HMIGZQpdF6jP
    7fkkVMROYvHUOJ8kknpKE7FiaNNpH7jK1qNxOYhLeiioX0LYrdmTvdTWHrSKZc82ZmlDjpKc4hUx
    VtTXMAysw7CzIdREPom/vJklnKLvZt+Wk5AEF5V5YKnuT3pY+fjVMZ56GtTEeO/Er/oLk/n2xUK5
    fD5DAyW/9z0ygzwTbY5IuWXyDfYneL4nXwWOEgg37Z4+8mTH+ftTz2dl1x1KIlIR2xo0kxf9t8K+
    jlr13vwF1+QReMCSUycUsZ2Eep5XhjI+LG7G1bMSGqodZTIOXLkIy6+3iJ8Z/feIHlJ0ELBDyFbl
    Yy04Sf9LI148vJMsYenonkoWejWdMi8iCUTeaZydHJEUBU/RbNFLjCWa6NIUe9bfZgLiOOZkps54
    +/AL078ri/tGjo/5UGvezSmwrEoWJyqrJt2M69N2oVDLJcHeo2bUYPtFC2Kfb2je58JrJ+llifdg
    rAsxbnHXiXyVimUAEQEAAQ==
device-key-sha3-384: EAD4DbLxK_kn0gzNCXOs3kd6DeMU3f-L6BEsSEuJGBqCORR0gXkdDxMbOm11mRFu
timestamp: 2016-08-24T21:55:00Z
sign-key-sha3-384: Jv8_JiHiIzJVcO9M55pPdqSDWUvuhfDIBJUS-3VW7F_idjix7Ffn5qMxB21ZQuij

AXNpZw=`

	exDeviceSessionRequest = `type: device-session-request
brand-id: my-brand
model: baz-3000
serial: 9999
nonce: @NONCE@
timestamp: @TS@
sign-key-sha3-384: Jv8_JiHiIzJVcO9M55pPdqSDWUvuhfDIBJUS-3VW7F_idjix7Ffn5qMxB21ZQuij

AXNpZw=`
)

type testDeviceAssertions struct {
	nothing bool
}

func (da *testDeviceAssertions) Model() (*asserts.Model, error) {
	if da.nothing {
		return nil, state.ErrNoState
	}
	a, err := asserts.Decode([]byte(exModel))
	if err != nil {
		return nil, err
	}
	return a.(*asserts.Model), nil
}

func (da *testDeviceAssertions) Serial() (*asserts.Serial, error) {
	if da.nothing {
		return nil, state.ErrNoState
	}
	a, err := asserts.Decode([]byte(exSerial))
	if err != nil {
		return nil, err
	}
	return a.(*asserts.Serial), nil
}

func (da *testDeviceAssertions) DeviceSessionRequest(nonce string) (*asserts.DeviceSessionRequest, *asserts.Serial, error) {
	if da.nothing {
		return nil, nil, state.ErrNoState
	}
	ex := strings.Replace(exDeviceSessionRequest, "@NONCE@", nonce, 1)
	ex = strings.Replace(ex, "@TS@", time.Now().Format(time.RFC3339), 1)
	a1, err := asserts.Decode([]byte(ex))
	if err != nil {
		return nil, nil, err
	}

	a2, err := asserts.Decode([]byte(exSerial))
	if err != nil {
		return nil, nil, err
	}
	return a1.(*asserts.DeviceSessionRequest), a2.(*asserts.Serial), nil
}

func (as *authSuite) TestAuthContextMissingDeviceAssertions(c *C) {
	// no assertions in state
	authContext := auth.NewAuthContext(as.state, &testDeviceAssertions{nothing: true})

	_, _, err := authContext.DeviceSessionRequest("NONCE")
	c.Check(err, Equals, auth.ErrNoSerial)

	storeID, err := authContext.StoreID("fallback")
	c.Assert(err, IsNil)
	c.Check(storeID, Equals, "fallback")
}

func (as *authSuite) TestAuthContextWithDeviceAssertions(c *C) {
	// having assertions in state
	authContext := auth.NewAuthContext(as.state, &testDeviceAssertions{})

	req, serial, err := authContext.DeviceSessionRequest("NONCE-1")
	c.Assert(err, IsNil)
	c.Check(strings.Contains(string(req), "nonce: NONCE-1\n"), Equals, true)
	c.Check(strings.Contains(string(req), "serial: 9999\n"), Equals, true)
	c.Check(strings.Contains(string(serial), "serial: 9999\n"), Equals, true)

	storeID, err := authContext.StoreID("store-id")
	c.Assert(err, IsNil)
	c.Check(storeID, Equals, "my-brand-store-id")
}
