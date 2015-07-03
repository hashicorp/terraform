package docker

import (
	"fmt"
	"strings"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerImageCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)
	apiImage, err := findImage(d, client)
	if err != nil {
		return fmt.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.SetId(apiImage.ID + d.Get("name").(string))
	d.Set("latest", apiImage.ID)

	return nil
}

func resourceDockerImageRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)
	apiImage, err := findImage(d, client)
	if err != nil {
		return fmt.Errorf("Unable to read Docker image into resource: %s", err)
	}

	d.Set("latest", apiImage.ID)

	return nil
}

func resourceDockerImageUpdate(d *schema.ResourceData, meta interface{}) error {
	// We need to re-read in case switching parameters affects
	// the value of "latest" or others

	return resourceDockerImageRead(d, meta)
}

func resourceDockerImageDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func fetchLocalImages(data *Data, client *dc.Client) error {
	images, err := client.ListImages(dc.ListImagesOptions{All: false})
	if err != nil {
		return fmt.Errorf("Unable to list Docker images: %s", err)
	}

	if data.DockerImages == nil {
		data.DockerImages = make(map[string]*dc.APIImages)
	}

	// Docker uses different nomenclatures in different places...sometimes a short
	// ID, sometimes long, etc. So we store both in the map so we can always find
	// the same image object. We store the tags, too.
	for i, image := range images {
		data.DockerImages[image.ID[:12]] = &images[i]
		data.DockerImages[image.ID] = &images[i]
		for _, repotag := range image.RepoTags {
			data.DockerImages[repotag] = &images[i]
		}
	}

	return nil
}

func pullImage(data *Data, client *dc.Client, image string) error {
	// TODO: Test local registry handling. It should be working
	// based on the code that was ported over

	pullOpts := dc.PullImageOptions{}

	splitImageName := strings.Split(image, ":")
	switch len(splitImageName) {

	// It's in registry:port/username/repo:tag or registry:port/repo:tag format
	case 3:
		splitPortRepo := strings.Split(splitImageName[1], "/")
		pullOpts.Registry = splitImageName[0] + ":" + splitPortRepo[0]
		pullOpts.Tag = splitImageName[2]
		pullOpts.Repository = pullOpts.Registry + "/" + strings.Join(splitPortRepo[1:], "/")

	// It's either registry:port/username/repo, registry:port/repo,
	// or repo:tag with default registry
	case 2:
		splitPortRepo := strings.Split(splitImageName[1], "/")
		switch len(splitPortRepo) {
		// repo:tag
		case 1:
			pullOpts.Repository = splitImageName[0]
			pullOpts.Tag = splitImageName[1]

		// registry:port/username/repo or registry:port/repo
		default:
			pullOpts.Registry = splitImageName[0] + ":" + splitPortRepo[0]
			pullOpts.Repository = pullOpts.Registry + "/" + strings.Join(splitPortRepo[1:], "/")
			pullOpts.Tag = "latest"
		}

	// Plain username/repo or repo
	default:
		pullOpts.Repository = image
	}

	if err := client.PullImage(pullOpts, dc.AuthConfiguration{}); err != nil {
		return fmt.Errorf("Error pulling image %s: %s\n", image, err)
	}

	return fetchLocalImages(data, client)
}

func getImageTag(image string) string {
	splitImageName := strings.Split(image, ":")
	switch {

	// It's in registry:port/repo:tag format
	case len(splitImageName) == 3:
		return splitImageName[2]

	// It's either registry:port/repo or repo:tag with default registry
	case len(splitImageName) == 2:
		splitPortRepo := strings.Split(splitImageName[1], "/")
		if len(splitPortRepo) == 2 {
			return ""
		} else {
			return splitImageName[1]
		}
	}

	return ""
}

func findImage(d *schema.ResourceData, client *dc.Client) (*dc.APIImages, error) {
	var data Data
	if err := fetchLocalImages(&data, client); err != nil {
		return nil, err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return nil, fmt.Errorf("Empty image name is not allowed")
	}

	searchLocal := func() *dc.APIImages {
		if apiImage, ok := data.DockerImages[imageName]; ok {
			return apiImage
		}
		if apiImage, ok := data.DockerImages[imageName+":latest"]; ok {
			imageName = imageName + ":latest"
			return apiImage
		}
		return nil
	}

	foundImage := searchLocal()

	if d.Get("keep_updated").(bool) || foundImage == nil {
		if err := pullImage(&data, client, imageName); err != nil {
			return nil, fmt.Errorf("Unable to pull image %s: %s", imageName, err)
		}
	}

	foundImage = searchLocal()
	if foundImage != nil {
		return foundImage, nil
	}

	return nil, fmt.Errorf("Unable to find or pull image %s", imageName)
}
