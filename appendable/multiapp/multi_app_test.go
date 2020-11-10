/*
Copyright 2019-2020 vChain, Inc.

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
package multiapp

import (
	"os"
	"testing"

	"codenotary.io/immudb-v2/appendable"
	"github.com/stretchr/testify/require"
)

func TestMultiApp(t *testing.T) {
	md := appendable.NewMetadata(nil)
	md.PutInt("mkey1", 1)

	a, err := Open("testdata", DefaultOptions().SetMetadata(md.Bytes()))
	defer os.RemoveAll("testdata")
	require.NoError(t, err)

	sz, err := a.Size()
	require.NoError(t, err)
	require.Equal(t, int64(0), sz)

	require.Equal(t, appendable.DefaultCompressionFormat, a.CompressionFormat())
	require.Equal(t, appendable.DefaultCompressionLevel, a.CompressionLevel())

	err = a.SetOffset(0)
	require.NoError(t, err)

	require.Equal(t, int64(0), a.Offset())

	mkey1, found := appendable.NewMetadata(a.Metadata()).GetInt("mkey1")
	require.True(t, found)
	require.Equal(t, 1, mkey1)

	_, _, err = a.Append(nil)
	require.Error(t, ErrIllegalArguments, err)

	off, n, err := a.Append([]byte{})
	require.NoError(t, err)
	require.Equal(t, int64(0), off)
	require.Equal(t, 0, n)

	off, n, err = a.Append([]byte{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, int64(0), off)
	require.Equal(t, 3, n)

	off, n, err = a.Append([]byte{4, 5, 6, 7, 8, 9, 10})
	require.NoError(t, err)
	require.Equal(t, int64(3), off)
	require.Equal(t, 7, n)

	err = a.Flush()
	require.NoError(t, err)

	bs := make([]byte, 3)
	n, err = a.ReadAt(bs, 0)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3}, bs)

	bs = make([]byte, 4)
	n, err = a.ReadAt(bs, 6)
	require.NoError(t, err)
	require.Equal(t, []byte{7, 8, 9, 10}, bs)

	err = a.Sync()
	require.NoError(t, err)

	err = a.Close()
	require.NoError(t, err)
}

func TestMultiAppReOpening(t *testing.T) {
	a, err := Open("testdata", DefaultOptions().SetFileSize(1))
	defer os.RemoveAll("testdata")
	require.NoError(t, err)

	off, n, err := a.Append([]byte{1, 2})
	require.NoError(t, err)
	require.Equal(t, int64(0), off)
	require.Equal(t, 2, n)

	off, n, err = a.Append([]byte{3})
	require.NoError(t, err)
	require.Equal(t, int64(2), off)
	require.Equal(t, 1, n)

	err = a.Close()
	require.NoError(t, err)

	a, err = Open("testdata", DefaultOptions().SetReadOnly(true))
	require.NoError(t, err)

	sz, err := a.Size()
	require.NoError(t, err)
	require.Equal(t, int64(3), sz)

	bs := make([]byte, 3)
	n, err = a.ReadAt(bs, 0)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3}, bs)

	_, _, err = a.Append([]byte{})
	require.Error(t, ErrReadOnly, err)

	err = a.Flush()
	require.Error(t, ErrReadOnly, err)

	err = a.Sync()
	require.Error(t, ErrReadOnly, err)

	err = a.Close()
	require.NoError(t, err)
}

func TestMultiAppEdgeCases(t *testing.T) {
	_, err := Open("testdata", nil)
	require.Error(t, ErrIllegalArguments, err)

	_, err = Open("multi_app_test.go", DefaultOptions())
	require.Error(t, ErrorPathIsNotADirectory, err)

	_, err = Open("testdata", DefaultOptions().SetReadOnly(true))
	require.Error(t, err)

	a, err := Open("testdata", DefaultOptions())
	defer os.RemoveAll("testdata")
	require.NoError(t, err)

	_, err = a.ReadAt(nil, 0)
	require.Error(t, ErrIllegalArguments, err)

	err = a.Close()
	require.NoError(t, err)

	_, err = a.Size()
	require.Error(t, ErrAlreadyClosed, err)

	err = a.SetOffset(0)
	require.Error(t, ErrAlreadyClosed, err)

	_, _, err = a.Append([]byte{})
	require.Error(t, ErrAlreadyClosed, err)

	_, err = a.ReadAt([]byte{}, 0)
	require.Error(t, ErrIllegalArguments, err)

	err = a.Flush()
	require.Error(t, ErrAlreadyClosed, err)

	err = a.Sync()
	require.Error(t, ErrAlreadyClosed, err)

	err = a.Close()
	require.Error(t, ErrAlreadyClosed, err)
}

func TestMultiAppCompression(t *testing.T) {
	a, err := Open("testdata", DefaultOptions().SetCompressionFormat(appendable.ZLibCompression))
	defer os.RemoveAll("testdata")
	require.NoError(t, err)

	off, _, err := a.Append([]byte{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, int64(0), off)

	err = a.Flush()
	require.NoError(t, err)

	bs := make([]byte, 3)
	_, err = a.ReadAt(bs, 0)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3}, bs)

	err = a.Close()
	require.NoError(t, err)
}

func TestOptions(t *testing.T) {
	opts := &Options{}
	require.True(t, opts.SetReadOnly(true).readOnly)
	require.True(t, opts.SetSynced(true).synced)
	require.Equal(t, DefaultFileMode, opts.SetFileMode(DefaultFileMode).fileMode)
	require.Equal(t, appendable.DefaultCompressionFormat, opts.SetCompressionFormat(appendable.DefaultCompressionFormat).compressionFormat)
	require.Equal(t, appendable.DefaultCompressionLevel, opts.SetCompresionLevel(appendable.DefaultCompressionLevel).compressionLevel)
	require.Equal(t, []byte{1, 2, 3, 4}, opts.SetMetadata([]byte{1, 2, 3, 4}).metadata)
	require.Equal(t, DefaultFileSize, opts.SetFileSize(DefaultFileSize).fileSize)
	require.Equal(t, "aof", opts.SetFileExt("aof").fileExt)
	require.Equal(t, DefaultMaxOpenedFiles, opts.SetMaxOpenedFiles(DefaultMaxOpenedFiles).maxOpenedFiles)
}
