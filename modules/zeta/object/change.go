// Copyright 2018 Sourced Technologies, S.L.
// SPDX-License-Identifier: Apache-2.0

package object

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/antgroup/hugescm/modules/merkletrie"
)

// Change values represent a detected change between two git trees.  For
// modifications, From is the original status of the node and To is its
// final status.  For insertions, From is the zero value and for
// deletions To is the zero value.
type Change struct {
	From ChangeEntry
	To   ChangeEntry
}

var empty ChangeEntry

// Action returns the kind of action represented by the change, an
// insertion, a deletion or a modification.
func (c *Change) Action() (merkletrie.Action, error) {
	if c.From.Equal(&empty) && c.To.Equal(&empty) {
		return merkletrie.Action(0),
			fmt.Errorf("malformed change: empty from and to")
	}

	if c.From.Equal(&empty) {
		return merkletrie.Insert, nil
	}

	if c.To.Equal(&empty) {
		return merkletrie.Delete, nil
	}

	return merkletrie.Modify, nil
}

// Files returns the files before and after a change.
// For insertions from will be nil. For deletions to will be nil.
func (c *Change) Files() (from, to *File, err error) {
	action, err := c.Action()
	if err != nil {
		return
	}

	if action == merkletrie.Insert || action == merkletrie.Modify {
		if !c.To.TreeEntry.Mode.IsFile() {
			return nil, nil, nil
		}
		e := &c.To.TreeEntry
		to = newFile(e.Name, e.Mode, e.Hash, e.Size, c.To.Tree.b)
	}

	if action == merkletrie.Delete || action == merkletrie.Modify {
		if !c.From.TreeEntry.Mode.IsFile() {
			return nil, nil, nil
		}
		e := &c.From.TreeEntry
		from = newFile(e.Name, e.Mode, e.Hash, e.Size, c.From.Tree.b)
	}
	return
}

func (c *Change) String() string {
	action, err := c.Action()
	if err != nil {
		return "malformed change"
	}

	return fmt.Sprintf("<Action: %s, Path: %s>", action, c.name())
}

// Patch returns a Patch with all the file changes in chunks. This
// representation can be used to create several diff outputs.
func (c *Change) Patch(codecvt bool) (*Patch, error) {
	return c.PatchContext(context.Background(), codecvt)
}

// Patch returns a Patch with all the file changes in chunks. This
// representation can be used to create several diff outputs.
// If context expires, an non-nil error will be returned
// Provided context must be non-nil
func (c *Change) PatchContext(ctx context.Context, codecvt bool) (*Patch, error) {
	return getPatchContext(ctx, "", codecvt, c)
}

func (c *Change) name() string {
	if !c.From.Equal(&empty) {
		return c.From.Name
	}

	return c.To.Name
}

// ChangeEntry values represent a node that has suffered a change.
type ChangeEntry struct {
	// Full path of the node using "/" as separator.
	Name string
	// Parent tree of the node that has changed.
	Tree *Tree
	// The entry of the node.
	TreeEntry TreeEntry
}

func (e *ChangeEntry) IsFragments() bool {
	return e.TreeEntry.IsFragments()
}

func (e *ChangeEntry) Equal(o *ChangeEntry) bool {
	return e.Name == o.Name && e.Tree.Equal(o.Tree) && e.TreeEntry.Equal(&o.TreeEntry)
}

// Changes represents a collection of changes between two git trees.
// Implements sort.Interface lexicographically over the path of the
// changed files.
type Changes []*Change

func (c Changes) Len() int {
	return len(c)
}

func (c Changes) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Changes) Less(i, j int) bool {
	return strings.Compare(c[i].name(), c[j].name()) < 0
}

func (c Changes) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	comma := ""
	for _, v := range c {
		buffer.WriteString(comma)
		buffer.WriteString(v.String())
		comma = ", "
	}
	buffer.WriteString("]")

	return buffer.String()
}

// Patch returns a Patch with all the changes in chunks. This
// representation can be used to create several diff outputs.
func (c Changes) Patch(codecvt bool) (*Patch, error) {
	return c.PatchContext(context.Background(), codecvt)
}

// Patch returns a Patch with all the changes in chunks. This
// representation can be used to create several diff outputs.
// If context expires, an non-nil error will be returned
// Provided context must be non-nil
func (c Changes) PatchContext(ctx context.Context, codecvt bool) (*Patch, error) {
	return getPatchContext(ctx, "", codecvt, c...)
}
