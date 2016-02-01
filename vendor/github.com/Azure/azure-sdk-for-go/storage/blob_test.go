package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	chk "github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/gopkg.in/check.v1"
)

type StorageBlobSuite struct{}

var _ = chk.Suite(&StorageBlobSuite{})

const testContainerPrefix = "zzzztest-"

func getBlobClient(c *chk.C) BlobStorageClient {
	return getBasicClient(c).GetBlobService()
}

func (s *StorageBlobSuite) Test_pathForContainer(c *chk.C) {
	c.Assert(pathForContainer("foo"), chk.Equals, "/foo")
}

func (s *StorageBlobSuite) Test_pathForBlob(c *chk.C) {
	c.Assert(pathForBlob("foo", "blob"), chk.Equals, "/foo/blob")
}

func (s *StorageBlobSuite) Test_blobSASStringToSign(c *chk.C) {
	_, err := blobSASStringToSign("2012-02-12", "CS", "SE", "SP")
	c.Assert(err, chk.NotNil) // not implemented SAS for versions earlier than 2013-08-15

	out, err := blobSASStringToSign("2013-08-15", "CS", "SE", "SP")
	c.Assert(err, chk.IsNil)
	c.Assert(out, chk.Equals, "SP\n\nSE\nCS\n\n2013-08-15\n\n\n\n\n")
}

func (s *StorageBlobSuite) TestGetBlobSASURI(c *chk.C) {
	api, err := NewClient("foo", "YmFy", DefaultBaseURL, "2013-08-15", true)
	c.Assert(err, chk.IsNil)
	cli := api.GetBlobService()
	expiry := time.Time{}

	expectedParts := url.URL{
		Scheme: "https",
		Host:   "foo.blob.core.windows.net",
		Path:   "container/name",
		RawQuery: url.Values{
			"sv":  {"2013-08-15"},
			"sig": {"/OXG7rWh08jYwtU03GzJM0DHZtidRGpC6g69rSGm3I0="},
			"sr":  {"b"},
			"sp":  {"r"},
			"se":  {"0001-01-01T00:00:00Z"},
		}.Encode()}

	u, err := cli.GetBlobSASURI("container", "name", expiry, "r")
	c.Assert(err, chk.IsNil)
	sasParts, err := url.Parse(u)
	c.Assert(err, chk.IsNil)
	c.Assert(expectedParts.String(), chk.Equals, sasParts.String())
	c.Assert(expectedParts.Query(), chk.DeepEquals, sasParts.Query())
}

func (s *StorageBlobSuite) TestBlobSASURICorrectness(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	blob := randString(20)
	body := []byte(randString(100))
	expiry := time.Now().UTC().Add(time.Hour)
	permissions := "r"

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.DeleteContainer(cnt)

	c.Assert(cli.putSingleBlockBlob(cnt, blob, body), chk.IsNil)

	sasURI, err := cli.GetBlobSASURI(cnt, blob, expiry, permissions)
	c.Assert(err, chk.IsNil)

	resp, err := http.Get(sasURI)
	c.Assert(err, chk.IsNil)

	blobResp, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, chk.IsNil)

	c.Assert(resp.StatusCode, chk.Equals, http.StatusOK)
	c.Assert(len(blobResp), chk.Equals, len(body))
}

func (s *StorageBlobSuite) TestListContainersPagination(c *chk.C) {
	cli := getBlobClient(c)
	c.Assert(deleteTestContainers(cli), chk.IsNil)

	const n = 5
	const pageSize = 2

	// Create test containers
	created := []string{}
	for i := 0; i < n; i++ {
		name := randContainer()
		c.Assert(cli.CreateContainer(name, ContainerAccessTypePrivate), chk.IsNil)
		created = append(created, name)
	}
	sort.Strings(created)

	// Defer test container deletions
	defer func() {
		var wg sync.WaitGroup
		for _, cnt := range created {
			wg.Add(1)
			go func(name string) {
				c.Assert(cli.DeleteContainer(name), chk.IsNil)
				wg.Done()
			}(cnt)
		}
		wg.Wait()
	}()

	// Paginate results
	seen := []string{}
	marker := ""
	for {
		resp, err := cli.ListContainers(ListContainersParameters{
			Prefix:     testContainerPrefix,
			MaxResults: pageSize,
			Marker:     marker})
		c.Assert(err, chk.IsNil)

		containers := resp.Containers
		if len(containers) > pageSize {
			c.Fatalf("Got a bigger page. Expected: %d, got: %d", pageSize, len(containers))
		}

		for _, c := range containers {
			seen = append(seen, c.Name)
		}

		marker = resp.NextMarker
		if marker == "" || len(containers) == 0 {
			break
		}
	}

	c.Assert(seen, chk.DeepEquals, created)
}

func (s *StorageBlobSuite) TestContainerExists(c *chk.C) {
	cnt := randContainer()
	cli := getBlobClient(c)
	ok, err := cli.ContainerExists(cnt)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, false)

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypeBlob), chk.IsNil)
	defer cli.DeleteContainer(cnt)

	ok, err = cli.ContainerExists(cnt)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, true)
}

func (s *StorageBlobSuite) TestCreateContainerDeleteContainer(c *chk.C) {
	cnt := randContainer()
	cli := getBlobClient(c)
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	c.Assert(cli.DeleteContainer(cnt), chk.IsNil)
}

func (s *StorageBlobSuite) TestCreateContainerIfNotExists(c *chk.C) {
	cnt := randContainer()
	cli := getBlobClient(c)
	defer cli.DeleteContainer(cnt)

	// First create
	ok, err := cli.CreateContainerIfNotExists(cnt, ContainerAccessTypePrivate)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, true)

	// Second create, should not give errors
	ok, err = cli.CreateContainerIfNotExists(cnt, ContainerAccessTypePrivate)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, false)
}

func (s *StorageBlobSuite) TestDeleteContainerIfExists(c *chk.C) {
	cnt := randContainer()
	cli := getBlobClient(c)

	// Nonexisting container
	c.Assert(cli.DeleteContainer(cnt), chk.NotNil)

	ok, err := cli.DeleteContainerIfExists(cnt)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, false)

	// Existing container
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	ok, err = cli.DeleteContainerIfExists(cnt)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, true)
}

func (s *StorageBlobSuite) TestBlobExists(c *chk.C) {
	cnt := randContainer()
	blob := randString(20)
	cli := getBlobClient(c)

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypeBlob), chk.IsNil)
	defer cli.DeleteContainer(cnt)
	c.Assert(cli.putSingleBlockBlob(cnt, blob, []byte("Hello!")), chk.IsNil)
	defer cli.DeleteBlob(cnt, blob)

	ok, err := cli.BlobExists(cnt, blob+".foo")
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, false)

	ok, err = cli.BlobExists(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, true)
}

func (s *StorageBlobSuite) TestGetBlobURL(c *chk.C) {
	api, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	cli := api.GetBlobService()

	c.Assert(cli.GetBlobURL("c", "nested/blob"), chk.Equals, "https://foo.blob.core.windows.net/c/nested/blob")
	c.Assert(cli.GetBlobURL("", "blob"), chk.Equals, "https://foo.blob.core.windows.net/$root/blob")
	c.Assert(cli.GetBlobURL("", "nested/blob"), chk.Equals, "https://foo.blob.core.windows.net/$root/nested/blob")
}

func (s *StorageBlobSuite) TestBlobCopy(c *chk.C) {
	if testing.Short() {
		c.Skip("skipping blob copy in short mode, no SLA on async operation")
	}

	cli := getBlobClient(c)
	cnt := randContainer()
	src := randString(20)
	dst := randString(20)
	body := []byte(randString(1024))

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	c.Assert(cli.putSingleBlockBlob(cnt, src, body), chk.IsNil)
	defer cli.DeleteBlob(cnt, src)

	c.Assert(cli.CopyBlob(cnt, dst, cli.GetBlobURL(cnt, src)), chk.IsNil)
	defer cli.DeleteBlob(cnt, dst)

	blobBody, err := cli.GetBlob(cnt, dst)
	c.Assert(err, chk.IsNil)

	b, err := ioutil.ReadAll(blobBody)
	defer blobBody.Close()
	c.Assert(err, chk.IsNil)
	c.Assert(b, chk.DeepEquals, body)
}

func (s *StorageBlobSuite) TestDeleteBlobIfExists(c *chk.C) {
	cnt := randContainer()
	blob := randString(20)

	cli := getBlobClient(c)
	c.Assert(cli.DeleteBlob(cnt, blob), chk.NotNil)

	ok, err := cli.DeleteBlobIfExists(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(ok, chk.Equals, false)
}

func (s *StorageBlobSuite) TestGetBlobProperties(c *chk.C) {
	cnt := randContainer()
	blob := randString(20)
	contents := randString(64)

	cli := getBlobClient(c)
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.DeleteContainer(cnt)

	// Nonexisting blob
	_, err := cli.GetBlobProperties(cnt, blob)
	c.Assert(err, chk.NotNil)

	// Put the blob
	c.Assert(cli.putSingleBlockBlob(cnt, blob, []byte(contents)), chk.IsNil)

	// Get blob properties
	props, err := cli.GetBlobProperties(cnt, blob)
	c.Assert(err, chk.IsNil)

	c.Assert(props.ContentLength, chk.Equals, int64(len(contents)))
	c.Assert(props.BlobType, chk.Equals, BlobTypeBlock)
}

func (s *StorageBlobSuite) TestListBlobsPagination(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.DeleteContainer(cnt)

	blobs := []string{}
	const n = 5
	const pageSize = 2
	for i := 0; i < n; i++ {
		name := randString(20)
		c.Assert(cli.putSingleBlockBlob(cnt, name, []byte("Hello, world!")), chk.IsNil)
		blobs = append(blobs, name)
	}
	sort.Strings(blobs)

	// Paginate
	seen := []string{}
	marker := ""
	for {
		resp, err := cli.ListBlobs(cnt, ListBlobsParameters{
			MaxResults: pageSize,
			Marker:     marker})
		c.Assert(err, chk.IsNil)

		for _, v := range resp.Blobs {
			seen = append(seen, v.Name)
		}

		marker = resp.NextMarker
		if marker == "" || len(resp.Blobs) == 0 {
			break
		}
	}

	// Compare
	c.Assert(seen, chk.DeepEquals, blobs)
}

func (s *StorageBlobSuite) TestGetAndSetMetadata(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	c.Assert(cli.putSingleBlockBlob(cnt, blob, []byte{}), chk.IsNil)

	m, err := cli.GetBlobMetadata(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(m, chk.Not(chk.Equals), nil)
	c.Assert(len(m), chk.Equals, 0)

	mPut := map[string]string{
		"foo":     "bar",
		"bar_baz": "waz qux",
	}

	err = cli.SetBlobMetadata(cnt, blob, mPut)
	c.Assert(err, chk.IsNil)

	m, err = cli.GetBlobMetadata(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Check(m, chk.DeepEquals, mPut)

	// Case munging

	mPutUpper := map[string]string{
		"Foo":     "different bar",
		"bar_BAZ": "different waz qux",
	}
	mExpectLower := map[string]string{
		"foo":     "different bar",
		"bar_baz": "different waz qux",
	}

	err = cli.SetBlobMetadata(cnt, blob, mPutUpper)
	c.Assert(err, chk.IsNil)

	m, err = cli.GetBlobMetadata(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Check(m, chk.DeepEquals, mExpectLower)
}

func (s *StorageBlobSuite) TestPutEmptyBlockBlob(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()

	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	c.Assert(cli.putSingleBlockBlob(cnt, blob, []byte{}), chk.IsNil)

	props, err := cli.GetBlobProperties(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(props.ContentLength, chk.Not(chk.Equals), 0)
}

func (s *StorageBlobSuite) TestGetBlobRange(c *chk.C) {
	cnt := randContainer()
	blob := randString(20)
	body := "0123456789"

	cli := getBlobClient(c)
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypeBlob), chk.IsNil)
	defer cli.DeleteContainer(cnt)

	c.Assert(cli.putSingleBlockBlob(cnt, blob, []byte(body)), chk.IsNil)
	defer cli.DeleteBlob(cnt, blob)

	// Read 1-3
	for _, r := range []struct {
		rangeStr string
		expected string
	}{
		{"0-", body},
		{"1-3", body[1 : 3+1]},
		{"3-", body[3:]},
	} {
		resp, err := cli.GetBlobRange(cnt, blob, r.rangeStr)
		c.Assert(err, chk.IsNil)
		blobBody, err := ioutil.ReadAll(resp)
		c.Assert(err, chk.IsNil)

		str := string(blobBody)
		c.Assert(str, chk.Equals, r.expected)
	}
}

func (s *StorageBlobSuite) TestCreateBlockBlobFromReader(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	name := randString(20)
	data := randBytes(8888)
	c.Assert(cli.CreateBlockBlobFromReader(cnt, name, uint64(len(data)), bytes.NewReader(data), nil), chk.IsNil)

	body, err := cli.GetBlob(cnt, name)
	c.Assert(err, chk.IsNil)
	gotData, err := ioutil.ReadAll(body)
	body.Close()

	c.Assert(err, chk.IsNil)
	c.Assert(gotData, chk.DeepEquals, data)
}

func (s *StorageBlobSuite) TestCreateBlockBlobFromReaderWithShortData(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	name := randString(20)
	data := randBytes(8888)
	err := cli.CreateBlockBlobFromReader(cnt, name, 9999, bytes.NewReader(data), nil)
	c.Assert(err, chk.Not(chk.IsNil))

	_, err = cli.GetBlob(cnt, name)
	// Upload was incomplete: blob should not have been created.
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *StorageBlobSuite) TestPutBlock(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	chunk := []byte(randString(1024))
	blockID := base64.StdEncoding.EncodeToString([]byte("foo"))
	c.Assert(cli.PutBlock(cnt, blob, blockID, chunk), chk.IsNil)
}

func (s *StorageBlobSuite) TestGetBlockList_PutBlockList(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	chunk := []byte(randString(1024))
	blockID := base64.StdEncoding.EncodeToString([]byte("foo"))

	// Put one block
	c.Assert(cli.PutBlock(cnt, blob, blockID, chunk), chk.IsNil)
	defer cli.deleteBlob(cnt, blob)

	// Get committed blocks
	committed, err := cli.GetBlockList(cnt, blob, BlockListTypeCommitted)
	c.Assert(err, chk.IsNil)

	if len(committed.CommittedBlocks) > 0 {
		c.Fatal("There are committed blocks")
	}

	// Get uncommitted blocks
	uncommitted, err := cli.GetBlockList(cnt, blob, BlockListTypeUncommitted)
	c.Assert(err, chk.IsNil)

	c.Assert(len(uncommitted.UncommittedBlocks), chk.Equals, 1)
	// Commit block list
	c.Assert(cli.PutBlockList(cnt, blob, []Block{{blockID, BlockStatusUncommitted}}), chk.IsNil)

	// Get all blocks
	all, err := cli.GetBlockList(cnt, blob, BlockListTypeAll)
	c.Assert(err, chk.IsNil)
	c.Assert(len(all.CommittedBlocks), chk.Equals, 1)
	c.Assert(len(all.UncommittedBlocks), chk.Equals, 0)

	// Verify the block
	thatBlock := all.CommittedBlocks[0]
	c.Assert(thatBlock.Name, chk.Equals, blockID)
	c.Assert(thatBlock.Size, chk.Equals, int64(len(chunk)))
}

func (s *StorageBlobSuite) TestCreateBlockBlob(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	c.Assert(cli.CreateBlockBlob(cnt, blob), chk.IsNil)

	// Verify
	blocks, err := cli.GetBlockList(cnt, blob, BlockListTypeAll)
	c.Assert(err, chk.IsNil)
	c.Assert(len(blocks.CommittedBlocks), chk.Equals, 0)
	c.Assert(len(blocks.UncommittedBlocks), chk.Equals, 0)
}

func (s *StorageBlobSuite) TestPutPageBlob(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	size := int64(10 * 1024 * 1024)
	c.Assert(cli.PutPageBlob(cnt, blob, size, nil), chk.IsNil)

	// Verify
	props, err := cli.GetBlobProperties(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(props.ContentLength, chk.Equals, size)
	c.Assert(props.BlobType, chk.Equals, BlobTypePage)
}

func (s *StorageBlobSuite) TestPutPagesUpdate(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	size := int64(10 * 1024 * 1024) // larger than we'll use
	c.Assert(cli.PutPageBlob(cnt, blob, size, nil), chk.IsNil)

	chunk1 := []byte(randString(1024))
	chunk2 := []byte(randString(512))

	// Append chunks
	c.Assert(cli.PutPage(cnt, blob, 0, int64(len(chunk1)-1), PageWriteTypeUpdate, chunk1), chk.IsNil)
	c.Assert(cli.PutPage(cnt, blob, int64(len(chunk1)), int64(len(chunk1)+len(chunk2)-1), PageWriteTypeUpdate, chunk2), chk.IsNil)

	// Verify contents
	out, err := cli.GetBlobRange(cnt, blob, fmt.Sprintf("%v-%v", 0, len(chunk1)+len(chunk2)-1))
	c.Assert(err, chk.IsNil)
	defer out.Close()
	blobContents, err := ioutil.ReadAll(out)
	c.Assert(err, chk.IsNil)
	c.Assert(blobContents, chk.DeepEquals, append(chunk1, chunk2...))
	out.Close()

	// Overwrite first half of chunk1
	chunk0 := []byte(randString(512))
	c.Assert(cli.PutPage(cnt, blob, 0, int64(len(chunk0)-1), PageWriteTypeUpdate, chunk0), chk.IsNil)

	// Verify contents
	out, err = cli.GetBlobRange(cnt, blob, fmt.Sprintf("%v-%v", 0, len(chunk1)+len(chunk2)-1))
	c.Assert(err, chk.IsNil)
	defer out.Close()
	blobContents, err = ioutil.ReadAll(out)
	c.Assert(err, chk.IsNil)
	c.Assert(blobContents, chk.DeepEquals, append(append(chunk0, chunk1[512:]...), chunk2...))
}

func (s *StorageBlobSuite) TestPutPagesClear(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	size := int64(10 * 1024 * 1024) // larger than we'll use
	c.Assert(cli.PutPageBlob(cnt, blob, size, nil), chk.IsNil)

	// Put 0-2047
	chunk := []byte(randString(2048))
	c.Assert(cli.PutPage(cnt, blob, 0, 2047, PageWriteTypeUpdate, chunk), chk.IsNil)

	// Clear 512-1023
	c.Assert(cli.PutPage(cnt, blob, 512, 1023, PageWriteTypeClear, nil), chk.IsNil)

	// Verify contents
	out, err := cli.GetBlobRange(cnt, blob, "0-2047")
	c.Assert(err, chk.IsNil)
	contents, err := ioutil.ReadAll(out)
	c.Assert(err, chk.IsNil)
	defer out.Close()
	c.Assert(contents, chk.DeepEquals, append(append(chunk[:512], make([]byte, 512)...), chunk[1024:]...))
}

func (s *StorageBlobSuite) TestGetPageRanges(c *chk.C) {
	cli := getBlobClient(c)
	cnt := randContainer()
	c.Assert(cli.CreateContainer(cnt, ContainerAccessTypePrivate), chk.IsNil)
	defer cli.deleteContainer(cnt)

	blob := randString(20)
	size := int64(10 * 1024 * 1024) // larger than we'll use
	c.Assert(cli.PutPageBlob(cnt, blob, size, nil), chk.IsNil)

	// Get page ranges on empty blob
	out, err := cli.GetPageRanges(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(len(out.PageList), chk.Equals, 0)

	// Add 0-512 page
	c.Assert(cli.PutPage(cnt, blob, 0, 511, PageWriteTypeUpdate, []byte(randString(512))), chk.IsNil)

	out, err = cli.GetPageRanges(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(len(out.PageList), chk.Equals, 1)

	// Add 1024-2048
	c.Assert(cli.PutPage(cnt, blob, 1024, 2047, PageWriteTypeUpdate, []byte(randString(1024))), chk.IsNil)

	out, err = cli.GetPageRanges(cnt, blob)
	c.Assert(err, chk.IsNil)
	c.Assert(len(out.PageList), chk.Equals, 2)
}

func deleteTestContainers(cli BlobStorageClient) error {
	for {
		resp, err := cli.ListContainers(ListContainersParameters{Prefix: testContainerPrefix})
		if err != nil {
			return err
		}
		if len(resp.Containers) == 0 {
			break
		}
		for _, c := range resp.Containers {
			err = cli.DeleteContainer(c.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b BlobStorageClient) putSingleBlockBlob(container, name string, chunk []byte) error {
	if len(chunk) > MaxBlobBlockSize {
		return fmt.Errorf("storage: provided chunk (%d bytes) cannot fit into single-block blob (max %d bytes)", len(chunk), MaxBlobBlockSize)
	}

	uri := b.client.getEndpoint(blobServiceName, pathForBlob(container, name), url.Values{})
	headers := b.client.getStandardHeaders()
	headers["x-ms-blob-type"] = string(BlobTypeBlock)
	headers["Content-Length"] = fmt.Sprintf("%v", len(chunk))

	resp, err := b.client.exec("PUT", uri, headers, bytes.NewReader(chunk))
	if err != nil {
		return err
	}
	return checkRespCode(resp.statusCode, []int{http.StatusCreated})
}

func randContainer() string {
	return testContainerPrefix + randString(32-len(testContainerPrefix))
}

func randString(n int) string {
	if n <= 0 {
		panic("negative number")
	}
	const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func randBytes(n int) []byte {
	data := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		panic(err)
	}
	return data
}
