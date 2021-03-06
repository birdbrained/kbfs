// Copyright 2017 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package kbfsgit

import (
	"github.com/keybase/kbfs/libfs"
	gogitcfg "gopkg.in/src-d/go-git.v4/config"
	format "gopkg.in/src-d/go-git.v4/plumbing/format/config"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
	"gopkg.in/src-d/go-git.v4/storage"
	"gopkg.in/src-d/go-git.v4/storage/filesystem"
)

// configWithoutRemotesStorer strips remotes from the config before
// writing them to disk, to work around a gcfg bug (used by go-git
// when reading configs from disk) that causes a freakout when it sees
// backslashes in git file URLs.
type configWithoutRemotesStorer struct {
	*filesystem.Storage
	cfg *gogitcfg.Config
}

func newConfigWithoutRemotesStorer(fs *libfs.FS) (
	*configWithoutRemotesStorer, error) {
	fsStorer, err := filesystem.NewStorage(fs)
	if err != nil {
		return nil, err
	}
	cfg, err := fsStorer.Config()
	if err != nil {
		return nil, err
	}
	return &configWithoutRemotesStorer{fsStorer, cfg}, nil
}

func (cwrs *configWithoutRemotesStorer) Init() error {
	return cwrs.Storage.Init()
}

func (cwrs *configWithoutRemotesStorer) Config() (*gogitcfg.Config, error) {
	return cwrs.cfg, nil
}

func (cwrs *configWithoutRemotesStorer) SetConfig(c *gogitcfg.Config) error {
	cwrs.cfg = c
	if len(c.Remotes) != 0 {
		// If there are remotes, we need to strip them out before
		// writing them out to disk.  Do that by making a copy of
		// everything but the remotes.  (Note that we can't just
		// Marshal+Unmarshal for a deep-copy, since Unmarshal is where
		// the gcfg bug is.)
		cCopy := gogitcfg.NewConfig()
		cCopy.Core = c.Core
		for k, v := range c.Submodules {
			v2 := *v
			cCopy.Submodules[k] = &v2
		}

		// Get the raw config so we don't lose any unsupported fields
		// from c, but clear out the remotes.
		_, err := c.Marshal()
		if err != nil {
			return err
		}
		s := c.Raw.Section("remote")
		s.Subsections = make(format.Subsections, 0)
		cCopy.Raw = c.Raw

		c = cCopy
	}
	return cwrs.Storage.SetConfig(c)
}

var _ storage.Storer = (*configWithoutRemotesStorer)(nil)
var _ storer.Initializer = (*configWithoutRemotesStorer)(nil)
