// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os/user"
	"path"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"github.com/tsuru/tsuru/fs/fstest"
	"gopkg.in/check.v1"
)

func (s *S) TestKeyAdd(c *check.C) {
	var stdout, stderr bytes.Buffer
	u, err := user.Current()
	c.Assert(err, check.IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	name := "my-key"
	expected := fmt.Sprintf("Key %q successfully added!\n", name)
	context := cmd.Context{
		Args:   []string{name, p},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/users/keys")
			key := r.FormValue("key") == "user-key"
			name := r.FormValue("name") == "my-key"
			force := r.FormValue("force") == "false"
			return method && url && key && name && force
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKeyAddStdin(c *check.C) {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBufferString("my powerful key")
	name := "my-key"
	expected := fmt.Sprintf("Key %q successfully added!\n", name)
	context := cmd.Context{
		Args:   []string{name, "-"},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  stdin,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			method := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/users/keys")
			key := r.FormValue("key") == "my powerful key"
			name := r.FormValue("name") == "my-key"
			force := r.FormValue("force") == "false"
			return method && url && key && name && force
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := KeyAdd{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAddKeyConfirmation(c *check.C) {
	var stdout, stderr bytes.Buffer
	u, err := user.Current()
	c.Assert(err, check.IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	name := "my-key"
	context := cmd.Context{
		Args:   []string{name, p},
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader("y\n"),
	}
	var calls int
	transport := cmdtest.MultiConditionalTransport{
		ConditionalTransports: []cmdtest.ConditionalTransport{
			{
				Transport: cmdtest.Transport{Message: "failed", Status: http.StatusConflict},
				CondFunc: func(r *http.Request) bool {
					calls++
					method := r.Method == "POST"
					url := strings.HasSuffix(r.URL.Path, "/users/keys")
					key := r.FormValue("key") == "user-key"
					name := r.FormValue("name") == "my-key"
					force := r.FormValue("force") == "false"
					return method && url && key && name && force
				},
			},
			{
				Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
				CondFunc: func(r *http.Request) bool {
					calls++
					method := r.Method == "POST"
					url := strings.HasSuffix(r.URL.Path, "/users/keys")
					key := r.FormValue("key") == "user-key"
					name := r.FormValue("name") == "my-key"
					force := r.FormValue("force") == "true"
					return method && url && key && name && force
				},
			},
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := `WARNING: key "my-key" already exists.
Do you want to replace it? (y/n) Key "my-key" successfully replaced!` + "\n"
	c.Assert(stdout.String(), check.Equals, expected)
	c.Assert(calls, check.Equals, 2)
}

func (s *S) TestAddKeyForceFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	u, err := user.Current()
	c.Assert(err, check.IsNil)
	p := path.Join(u.HomeDir, ".ssh", "id_rsa.pub")
	name := "my-key"
	expected := fmt.Sprintf("Key %q successfully added!\n", name)
	context := cmd.Context{
		Args:   []string{name, p},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			post := r.Method == "POST"
			url := strings.HasSuffix(r.URL.Path, "/users/keys")
			key := r.FormValue("key") == "user-key"
			name := r.FormValue("force") == "true"
			return post && url && key && name
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	command.Flags().Parse(true, []string{"-f"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKeyAddReturnsProperErrorIfTheGivenKeyFileDoesNotExist(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-key", "/unknown/key.pub"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fs := fstest.FileNotFoundFs{RecordingFs: fstest.RecordingFs{}}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, `file "/unknown/key.pub" doesn't exist`)
}

func (s *S) TestKeyAddFileSystemError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Args:   []string{"my-key", "/unknown/key.pub"},
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fs := fstest.FailureFs{
		RecordingFs: fstest.RecordingFs{},
		Err:         errors.New("what happened?"),
	}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	err := command.Run(&context, nil)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "what happened?")
}

func (s *S) TestKeyAddError(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{Args: []string{"my-key", "/tmp/id_rsa.pub"}, Stdout: &stdout, Stderr: &stderr}
	transport := cmdtest.Transport{
		Message: "something went wrong",
		Status:  http.StatusInternalServerError,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	fs := fstest.RecordingFs{FileContent: "user-key"}
	command := KeyAdd{keyReader: keyReader{fsystem: &fs}}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "something went wrong")
}

func (s *S) TestInfoKeyAdd(c *check.C) {
	c.Assert((&KeyAdd{}).Info(), check.NotNil)
}

func (s *S) TestKeyRemove(c *check.C) {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBufferString("y\n")
	keyName := "my-key"
	expected := fmt.Sprintf("Are you sure you want to remove key %q? (y/n) Key %q successfully removed!\n", keyName, keyName)
	context := cmd.Context{Args: []string{keyName}, Stdout: &stdout, Stderr: &stderr, Stdin: stdin}
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: "success", Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			method := r.Method == "DELETE"
			url := strings.HasSuffix(r.URL.Path, "/users/keys/my-key")
			return method && url
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := KeyRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKeyRemoveError(c *check.C) {
	var stdout, stderr bytes.Buffer
	stdin := bytes.NewBufferString("y\n")
	keyName := "my-key"
	context := cmd.Context{Args: []string{keyName}, Stdout: &stdout, Stderr: &stderr, Stdin: stdin}
	transport := cmdtest.Transport{
		Message: "something went wrong",
		Status:  http.StatusInternalServerError,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command := KeyRemove{}
	err := command.Run(&context, client)
	c.Assert(err, check.NotNil)
	c.Assert(err.Error(), check.Equals, "something went wrong")
}

func (s *S) TestInfoKeyRemove(c *check.C) {
	c.Assert((&KeyRemove{}).Info(), check.NotNil)
}

func (s *S) TestKeyList(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `+------+-----------------------------------------------------------------+
| Name | Content                                                         |
+------+-----------------------------------------------------------------+
| key1 | key1 content                                                    |
| key2 | key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 ... |
+------+-----------------------------------------------------------------+` + "\n"
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	key2Content := strings.Repeat("key2 ", 16)
	body := fmt.Sprintf(`{"key1":"key1 content","key2":%q}`, key2Content)
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: body, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/users/keys")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	var command KeyList
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestKeyListNoTruncate(c *check.C) {
	var stdout, stderr bytes.Buffer
	expected := `+------+----------------------------------------------------------------------------------+
| Name | Content                                                                          |
+------+----------------------------------------------------------------------------------+
| key1 | key1 content                                                                     |
+------+----------------------------------------------------------------------------------+
| key2 | key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2 key2  |
+------+----------------------------------------------------------------------------------+` + "\n"
	context := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	key2Content := strings.Repeat("key2 ", 16)
	body := fmt.Sprintf(`{"key1":"key1 content","key2":%q}`, key2Content)
	transport := cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: body, Status: http.StatusOK},
		CondFunc: func(r *http.Request) bool {
			return r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/users/keys")
		},
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	var command KeyList
	command.Flags().Parse(true, []string{"--no-truncate"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestInfoKeyList(c *check.C) {
	c.Assert((&KeyList{}).Info(), check.NotNil)
}
