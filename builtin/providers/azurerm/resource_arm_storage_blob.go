package azurerm

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmStorageBlob() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmStorageBlobCreate,
		Read:   resourceArmStorageBlobRead,
		Exists: resourceArmStorageBlobExists,
		Delete: resourceArmStorageBlobDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_account_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"storage_container_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageBlobType,
			},
			"size": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Default:      0,
				ValidateFunc: validateArmStorageBlobSize,
			},
			"source": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_uri"},
			},
			"source_uri": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source"},
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parallelism": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      8,
				ForceNew:     true,
				ValidateFunc: validateArmStorageBlobParallelism,
			},
			"attempts": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				ForceNew:     true,
				ValidateFunc: validateArmStorageBlobAttempts,
			},
		},
	}
}

func validateArmStorageBlobParallelism(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value <= 0 {
		errors = append(errors, fmt.Errorf("Blob Parallelism %q is invalid, must be greater than 0", value))
	}

	return
}

func validateArmStorageBlobAttempts(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value <= 0 {
		errors = append(errors, fmt.Errorf("Blob Attempts %q is invalid, must be greater than 0", value))
	}

	return
}

func validateArmStorageBlobSize(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)

	if value%512 != 0 {
		errors = append(errors, fmt.Errorf("Blob Size %q is invalid, must be a multiple of 512", value))
	}

	return
}

func validateArmStorageBlobType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	validTypes := map[string]struct{}{
		"block": struct{}{},
		"page":  struct{}{},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf("Blob type %q is invalid, must be %q or %q", value, "block", "page"))
	}
	return
}

func resourceArmStorageBlobCreate(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		return fmt.Errorf("Storage Account %q Not Found", storageAccountName)
	}

	name := d.Get("name").(string)
	blobType := d.Get("type").(string)
	cont := d.Get("storage_container_name").(string)
	sourceUri := d.Get("source_uri").(string)

	log.Printf("[INFO] Creating blob %q in storage account %q", name, storageAccountName)
	if sourceUri != "" {
		options := &storage.CopyOptions{}
		container := blobClient.GetContainerReference(cont)
		blob := container.GetBlobReference(name)
		err := blob.Copy(sourceUri, options)
		if err != nil {
			return fmt.Errorf("Error creating storage blob on Azure: %s", err)
		}
	} else {
		switch strings.ToLower(blobType) {
		case "block":
			options := &storage.PutBlobOptions{}
			container := blobClient.GetContainerReference(cont)
			blob := container.GetBlobReference(name)
			err := blob.CreateBlockBlob(options)
			if err != nil {
				return fmt.Errorf("Error creating storage blob on Azure: %s", err)
			}

			source := d.Get("source").(string)
			if source != "" {
				parallelism := d.Get("parallelism").(int)
				attempts := d.Get("attempts").(int)
				if err := resourceArmStorageBlobBlockUploadFromSource(cont, name, source, blobClient, parallelism, attempts); err != nil {
					return fmt.Errorf("Error creating storage blob on Azure: %s", err)
				}
			}
		case "page":
			source := d.Get("source").(string)
			if source != "" {
				parallelism := d.Get("parallelism").(int)
				attempts := d.Get("attempts").(int)
				if err := resourceArmStorageBlobPageUploadFromSource(cont, name, source, blobClient, parallelism, attempts); err != nil {
					return fmt.Errorf("Error creating storage blob on Azure: %s", err)
				}
			} else {
				size := int64(d.Get("size").(int))
				options := &storage.PutBlobOptions{}

				container := blobClient.GetContainerReference(cont)
				blob := container.GetBlobReference(name)
				blob.Properties.ContentLength = size
				err := blob.PutPageBlob(options)
				if err != nil {
					return fmt.Errorf("Error creating storage blob on Azure: %s", err)
				}
			}
		}
	}

	d.SetId(name)
	return resourceArmStorageBlobRead(d, meta)
}

type resourceArmStorageBlobPage struct {
	offset  int64
	section *io.SectionReader
}

func resourceArmStorageBlobPageUploadFromSource(container, name, source string, client *storage.BlobStorageClient, parallelism, attempts int) error {
	workerCount := parallelism * runtime.NumCPU()

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Error opening source file for upload %q: %s", source, err)
	}
	defer file.Close()

	blobSize, pageList, err := resourceArmStorageBlobPageSplit(file)
	if err != nil {
		return fmt.Errorf("Error splitting source file %q into pages: %s", source, err)
	}

	options := &storage.PutBlobOptions{}
	containerRef := client.GetContainerReference(container)
	blob := containerRef.GetBlobReference(name)
	blob.Properties.ContentLength = blobSize
	err = blob.PutPageBlob(options)
	if err != nil {
		return fmt.Errorf("Error creating storage blob on Azure: %s", err)
	}

	pages := make(chan resourceArmStorageBlobPage, len(pageList))
	errors := make(chan error, len(pageList))
	wg := &sync.WaitGroup{}
	wg.Add(len(pageList))

	total := int64(0)
	for _, page := range pageList {
		total += page.section.Size()
		pages <- page
	}
	close(pages)

	for i := 0; i < workerCount; i++ {
		go resourceArmStorageBlobPageUploadWorker(resourceArmStorageBlobPageUploadContext{
			container: container,
			name:      name,
			source:    source,
			blobSize:  blobSize,
			client:    client,
			pages:     pages,
			errors:    errors,
			wg:        wg,
			attempts:  attempts,
		})
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error while uploading source file %q: %s", source, <-errors)
	}

	return nil
}

func resourceArmStorageBlobPageSplit(file *os.File) (int64, []resourceArmStorageBlobPage, error) {
	const (
		minPageSize int64 = 4 * 1024
		maxPageSize int64 = 4 * 1024 * 1024
	)

	info, err := file.Stat()
	if err != nil {
		return int64(0), nil, fmt.Errorf("Could not stat file %q: %s", file.Name(), err)
	}

	blobSize := info.Size()
	if info.Size()%minPageSize != 0 {
		blobSize = info.Size() + (minPageSize - (info.Size() % minPageSize))
	}

	emptyPage := make([]byte, minPageSize)

	type byteRange struct {
		offset int64
		length int64
	}

	var nonEmptyRanges []byteRange
	var currentRange byteRange
	for i := int64(0); i < blobSize; i += minPageSize {
		pageBuf := make([]byte, minPageSize)
		_, err = file.ReadAt(pageBuf, i)
		if err != nil && err != io.EOF {
			return int64(0), nil, fmt.Errorf("Could not read chunk at %d: %s", i, err)
		}

		if bytes.Equal(pageBuf, emptyPage) {
			if currentRange.length != 0 {
				nonEmptyRanges = append(nonEmptyRanges, currentRange)
			}
			currentRange = byteRange{
				offset: i + minPageSize,
			}
		} else {
			currentRange.length += minPageSize
			if currentRange.length == maxPageSize || (currentRange.offset+currentRange.length == blobSize) {
				nonEmptyRanges = append(nonEmptyRanges, currentRange)
				currentRange = byteRange{
					offset: i + minPageSize,
				}
			}
		}
	}

	var pages []resourceArmStorageBlobPage
	for _, nonEmptyRange := range nonEmptyRanges {
		pages = append(pages, resourceArmStorageBlobPage{
			offset:  nonEmptyRange.offset,
			section: io.NewSectionReader(file, nonEmptyRange.offset, nonEmptyRange.length),
		})
	}

	return info.Size(), pages, nil
}

type resourceArmStorageBlobPageUploadContext struct {
	container string
	name      string
	source    string
	blobSize  int64
	client    *storage.BlobStorageClient
	pages     chan resourceArmStorageBlobPage
	errors    chan error
	wg        *sync.WaitGroup
	attempts  int
}

func resourceArmStorageBlobPageUploadWorker(ctx resourceArmStorageBlobPageUploadContext) {
	for page := range ctx.pages {
		start := page.offset
		end := page.offset + page.section.Size() - 1
		if end > ctx.blobSize-1 {
			end = ctx.blobSize - 1
		}
		size := end - start + 1

		chunk := make([]byte, size)
		_, err := page.section.Read(chunk)
		if err != nil && err != io.EOF {
			ctx.errors <- fmt.Errorf("Error reading source file %q at offset %d: %s", ctx.source, page.offset, err)
			ctx.wg.Done()
			continue
		}

		for x := 0; x < ctx.attempts; x++ {
			container := ctx.client.GetContainerReference(ctx.container)
			blob := container.GetBlobReference(ctx.name)
			blobRange := storage.BlobRange{
				Start: uint64(start),
				End:   uint64(end),
			}
			options := &storage.PutPageOptions{}
			reader := bytes.NewReader(chunk)
			err = blob.WriteRange(blobRange, reader, options)
			if err == nil {
				break
			}
		}
		if err != nil {
			ctx.errors <- fmt.Errorf("Error writing page at offset %d for file %q: %s", page.offset, ctx.source, err)
			ctx.wg.Done()
			continue
		}

		ctx.wg.Done()
	}
}

type resourceArmStorageBlobBlock struct {
	section *io.SectionReader
	id      string
}

func resourceArmStorageBlobBlockUploadFromSource(container, name, source string, client *storage.BlobStorageClient, parallelism, attempts int) error {
	workerCount := parallelism * runtime.NumCPU()

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Error opening source file for upload %q: %s", source, err)
	}
	defer file.Close()

	blockList, parts, err := resourceArmStorageBlobBlockSplit(file)
	if err != nil {
		return fmt.Errorf("Error reading and splitting source file for upload %q: %s", source, err)
	}

	wg := &sync.WaitGroup{}
	blocks := make(chan resourceArmStorageBlobBlock, len(parts))
	errors := make(chan error, len(parts))

	wg.Add(len(parts))
	for _, p := range parts {
		blocks <- p
	}
	close(blocks)

	for i := 0; i < workerCount; i++ {
		go resourceArmStorageBlobBlockUploadWorker(resourceArmStorageBlobBlockUploadContext{
			client:    client,
			source:    source,
			container: container,
			name:      name,
			blocks:    blocks,
			errors:    errors,
			wg:        wg,
			attempts:  attempts,
		})
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error while uploading source file %q: %s", source, <-errors)
	}

	containerReference := client.GetContainerReference(container)
	blobReference := containerReference.GetBlobReference(name)
	options := &storage.PutBlockListOptions{}
	err = blobReference.PutBlockList(blockList, options)
	if err != nil {
		return fmt.Errorf("Error updating block list for source file %q: %s", source, err)
	}

	return nil
}

func resourceArmStorageBlobBlockSplit(file *os.File) ([]storage.Block, []resourceArmStorageBlobBlock, error) {
	const (
		idSize          = 64
		blockSize int64 = 4 * 1024 * 1024
	)
	var parts []resourceArmStorageBlobBlock
	var blockList []storage.Block

	info, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("Error stating source file %q: %s", file.Name(), err)
	}

	for i := int64(0); i < info.Size(); i = i + blockSize {
		entropy := make([]byte, idSize)
		_, err = rand.Read(entropy)
		if err != nil {
			return nil, nil, fmt.Errorf("Error generating a random block ID for source file %q: %s", file.Name(), err)
		}

		sectionSize := blockSize
		remainder := info.Size() - i
		if remainder < blockSize {
			sectionSize = remainder
		}

		block := storage.Block{
			ID:     base64.StdEncoding.EncodeToString(entropy),
			Status: storage.BlockStatusUncommitted,
		}

		blockList = append(blockList, block)

		parts = append(parts, resourceArmStorageBlobBlock{
			id:      block.ID,
			section: io.NewSectionReader(file, i, sectionSize),
		})
	}

	return blockList, parts, nil
}

type resourceArmStorageBlobBlockUploadContext struct {
	client    *storage.BlobStorageClient
	container string
	name      string
	source    string
	attempts  int
	blocks    chan resourceArmStorageBlobBlock
	errors    chan error
	wg        *sync.WaitGroup
}

func resourceArmStorageBlobBlockUploadWorker(ctx resourceArmStorageBlobBlockUploadContext) {
	for block := range ctx.blocks {
		buffer := make([]byte, block.section.Size())

		_, err := block.section.Read(buffer)
		if err != nil {
			ctx.errors <- fmt.Errorf("Error reading source file %q: %s", ctx.source, err)
			ctx.wg.Done()
			continue
		}

		for i := 0; i < ctx.attempts; i++ {
			container := ctx.client.GetContainerReference(ctx.container)
			blob := container.GetBlobReference(ctx.name)
			options := &storage.PutBlockOptions{}
			err = blob.PutBlock(block.id, buffer, options)
			if err == nil {
				break
			}
		}
		if err != nil {
			ctx.errors <- fmt.Errorf("Error uploading block %q for source file %q: %s", block.id, ctx.source, err)
			ctx.wg.Done()
			continue
		}

		ctx.wg.Done()
	}
}

func resourceArmStorageBlobRead(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing blob %q from state", storageAccountName, d.Id())
		d.SetId("")
		return nil
	}

	exists, err := resourceArmStorageBlobExists(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		// Exists already removed this from state
		return nil
	}

	name := d.Get("name").(string)
	storageContainerName := d.Get("storage_container_name").(string)

	container := blobClient.GetContainerReference(storageContainerName)
	blob := container.GetBlobReference(name)
	url := blob.GetURL()
	if url == "" {
		log.Printf("[INFO] URL for %q is empty", name)
	}
	d.Set("url", url)

	return nil
}

func resourceArmStorageBlobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return false, err
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing blob %q from state", storageAccountName, d.Id())
		d.SetId("")
		return false, nil
	}

	name := d.Get("name").(string)
	storageContainerName := d.Get("storage_container_name").(string)

	log.Printf("[INFO] Checking for existence of storage blob %q.", name)
	container := blobClient.GetContainerReference(storageContainerName)
	blob := container.GetBlobReference(name)
	exists, err := blob.Exists()
	if err != nil {
		return false, fmt.Errorf("error testing existence of storage blob %q: %s", name, err)
	}

	if !exists {
		log.Printf("[INFO] Storage blob %q no longer exists, removing from state...", name)
		d.SetId("")
	}

	return exists, nil
}

func resourceArmStorageBlobDelete(d *schema.ResourceData, meta interface{}) error {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(resourceGroupName, storageAccountName)
	if err != nil {
		return err
	}
	if !accountExists {
		log.Printf("[INFO]Storage Account %q doesn't exist so the blob won't exist", storageAccountName)
		return nil
	}

	name := d.Get("name").(string)
	storageContainerName := d.Get("storage_container_name").(string)

	log.Printf("[INFO] Deleting storage blob %q", name)
	options := &storage.DeleteBlobOptions{}
	container := blobClient.GetContainerReference(storageContainerName)
	blob := container.GetBlobReference(name)
	_, err = blob.DeleteIfExists(options)
	if err != nil {
		return fmt.Errorf("Error deleting storage blob %q: %s", name, err)
	}

	d.SetId("")
	return nil
}
