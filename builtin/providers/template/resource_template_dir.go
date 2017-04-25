package template

import (
	"archive/tar"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDir() *schema.Resource {
	return &schema.Resource{
		Create: resourceTemplateDirCreate,
		Read:   resourceTemplateDirRead,
		Delete: resourceTemplateDirDelete,

		Schema: map[string]*schema.Schema{
			"source_dir": {
				Type:        schema.TypeString,
				Description: "Path to the directory where the files to template reside",
				Required:    true,
				ForceNew:    true,
			},
			"vars": {
				Type:         schema.TypeMap,
				Optional:     true,
				Default:      make(map[string]interface{}),
				Description:  "Variables to substitute",
				ValidateFunc: validateVarsAttribute,
				ForceNew:     true,
			},
			"destination_dir": {
				Type:        schema.TypeString,
				Description: "Path to the directory where the templated files will be written",
				Required:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceTemplateDirRead(d *schema.ResourceData, meta interface{}) error {
	sourceDir := d.Get("source_dir").(string)
	destinationDir := d.Get("destination_dir").(string)

	// If the output doesn't exist, mark the resource for creation.
	if _, err := os.Stat(destinationDir); os.IsNotExist(err) {
		d.SetId("")
		return nil
	}

	// If the combined hash of the input and output directories is different from
	// the stored one, mark the resource for re-creation.
	//
	// The output directory is technically enough for the general case, but by
	// hashing the input directory as well, we make development much easier: when
	// a developer modifies one of the input files, the generation is
	// re-triggered.
	hash, err := generateID(sourceDir, destinationDir)
	if err != nil {
		return err
	}
	if hash != d.Id() {
		d.SetId("")
		return nil
	}

	return nil
}

func resourceTemplateDirCreate(d *schema.ResourceData, meta interface{}) error {
	sourceDir := d.Get("source_dir").(string)
	destinationDir := d.Get("destination_dir").(string)
	vars := d.Get("vars").(map[string]interface{})

	// Always delete the output first, otherwise files that got deleted from the
	// input directory might still be present in the output afterwards.
	if err := resourceTemplateDirDelete(d, meta); err != nil {
		return err
	}

	// Create the destination directory and any other intermediate directories
	// leading to it.
	if _, err := os.Stat(destinationDir); err != nil {
		if err := os.MkdirAll(destinationDir, 0777); err != nil {
			return err
		}
	}

	// Recursively crawl the input files/directories and generate the output ones.
	err := filepath.Walk(sourceDir, func(p string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(sourceDir, p)
		return generateDirFile(p, path.Join(destinationDir, relPath), f, vars)
	})
	if err != nil {
		return err
	}

	// Compute ID.
	hash, err := generateID(sourceDir, destinationDir)
	if err != nil {
		return err
	}
	d.SetId(hash)

	return nil
}

func resourceTemplateDirDelete(d *schema.ResourceData, _ interface{}) error {
	d.SetId("")

	destinationDir := d.Get("destination_dir").(string)
	if _, err := os.Stat(destinationDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(destinationDir); err != nil {
		return fmt.Errorf("could not delete directory %q: %s", destinationDir, err)
	}

	return nil
}

func generateDirFile(sourceDir, destinationDir string, f os.FileInfo, vars map[string]interface{}) error {
	inputContent, _, err := pathorcontents.Read(sourceDir)
	if err != nil {
		return err
	}

	outputContent, err := execute(inputContent, vars)
	if err != nil {
		return templateRenderError(fmt.Errorf("failed to render %v: %v", sourceDir, err))
	}

	outputDir := path.Dir(destinationDir)
	if _, err := os.Stat(outputDir); err != nil {
		if err := os.MkdirAll(outputDir, 0777); err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(destinationDir, []byte(outputContent), f.Mode())
	if err != nil {
		return err
	}

	return nil
}

func generateID(sourceDir, destinationDir string) (string, error) {
	inputHash, err := generateDirHash(sourceDir)
	if err != nil {
		return "", err
	}
	outputHash, err := generateDirHash(destinationDir)
	if err != nil {
		return "", err
	}
	checksum := sha1.Sum([]byte(inputHash + outputHash))
	return hex.EncodeToString(checksum[:]), nil
}

func generateDirHash(directoryPath string) (string, error) {
	tarData, err := tarDir(directoryPath)
	if err != nil {
		return "", fmt.Errorf("could not generate output checksum: %s", err)
	}

	checksum := sha1.Sum(tarData)
	return hex.EncodeToString(checksum[:]), nil
}

func tarDir(directoryPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	writeFile := func(p string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var header *tar.Header
		var file *os.File

		header, err = tar.FileInfoHeader(f, f.Name())
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(directoryPath, p)
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		file, err = os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	}

	if err := filepath.Walk(directoryPath, writeFile); err != nil {
		return []byte{}, err
	}
	if err := tw.Flush(); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}
