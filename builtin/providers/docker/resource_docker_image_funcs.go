package docker

import (
	"fmt"
	"regexp"

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
	client := meta.(*dc.Client)
	err := removeImage(d, client)
	if err != nil {
		return fmt.Errorf("Unable to remove Docker image: %s", err)
	}
	d.SetId("")
	return nil
}

func searchLocalImages(data Data, imageName string) *dc.APIImages {
	if apiImage, ok := data.DockerImages[imageName]; ok {
		return apiImage
	}
	if apiImage, ok := data.DockerImages[imageName+":latest"]; ok {
		imageName = imageName + ":latest"
		return apiImage
	}
	return nil
}

func removeImage(d *schema.ResourceData, client *dc.Client) error {
	var data Data

	if keepLocally := d.Get("keep_locally").(bool); keepLocally {
		return nil
	}

	if err := fetchLocalImages(&data, client); err != nil {
		return err
	}

	imageName := d.Get("name").(string)
	if imageName == "" {
		return fmt.Errorf("Empty image name is not allowed")
	}

	foundImage := searchLocalImages(data, imageName)

	if foundImage != nil {
		err := client.RemoveImage(foundImage.ID)
		if err != nil {
			return err
		}
	}

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

func getAuthConfiguration(data *Data, registry string) (string, dc.AuthConfiguration) {
	authConfiguration := data.DockerRegistry.Configs[registry]

	if authConfiguration == (dc.AuthConfiguration{}) {
		if len(data.DockerRegistry.Configs) == 1 {
			for firstRegistry, firstAuthentication := range data.DockerRegistry.Configs {
				registry = firstRegistry
				authConfiguration = firstAuthentication
			}
		}
	}

	if authConfiguration == (dc.AuthConfiguration{}) {
		authConfiguration = data.DockerRegistry.Configs["http://"+registry]
	}

	if authConfiguration == (dc.AuthConfiguration{}) {
		authConfiguration = data.DockerRegistry.Configs["https://"+registry]
	}

	// If registry is not found, we try with the default one
	if authConfiguration == (dc.AuthConfiguration{}) {
		registry = "https://index.docker.io/v1/"

		authConfiguration = data.DockerRegistry.Configs["https://index.docker.io/v1/"]
	}

	// If registry is not found, we try with the default one
	if authConfiguration == (dc.AuthConfiguration{}) {
		registry = "https://hub.docker.com/"

		authConfiguration = data.DockerRegistry.Configs["https://hub.docker.com/"]
	}

	if authConfiguration == (dc.AuthConfiguration{}) {
		registry = ""
	}

	return registry, authConfiguration
}

func pullImage(data *Data, client *dc.Client, image string) error {
	r, err := regexp.Compile(`^(http[s]{0,1}:\/\/|)(([a-zA-Z0-9\-_\.]+)\/|([a-zA-Z0-9\-_\.]+:([0-9]{1,5}))\/|)((([a-zA-Z0-9\-_\.]+\/|)*)([a-zA-Z0-9\-_\.]+))(|:([a-zA-Z0-9-_\.]+))$`)

	if err != nil {
		return fmt.Errorf("Error pulling image due to regexp %s: %s\n", image, err)
	}

	pullOpts := dc.PullImageOptions{}

	splitImageName := r.FindStringSubmatch(image)
	// Example of splitted repository
	// [0] Original: 							http://test.dkr.ecr.us-east-1.amazonaws.com:80/foo/bar/nervous-system:latest
	// [1] Protocol:							http://
	// [2] Regsitry raw:					test.dkr.ecr.us-east-1.amazonaws.com:80/
	// [3] Registry:
	// [4] Registry (if port):		test.dkr.ecr.us-east-1.amazonaws.com:80
	// [5] Port: 		  						80
	// [6] Repository:						foo/bar/nervous-system
	// [7] Garbage:								foo/bar/
	// [8] Garbage: 							bar/
	// [9] Garbage: 							bar
	// [10] Garbage:							:latest
	// [11] Tag: 									latest

	// If registry as a port defined, we take this information
	if len(splitImageName[3]) == 0 {
		splitImageName[3] = splitImageName[4]
	}

	// If there is no tag, we set "latest" by default
	if len(splitImageName[11]) == 0 {
		splitImageName[11] = "latest"
	}

	pullOpts.Registry = splitImageName[3]
	pullOpts.Repository = splitImageName[2] + splitImageName[6]
	pullOpts.Tag = splitImageName[11]

	var authConfiguration dc.AuthConfiguration

	if data.DockerRegistry != nil {
		pullOpts.Registry, authConfiguration = getAuthConfiguration(data, pullOpts.Registry)
	}

	if err := client.PullImage(pullOpts, authConfiguration); err != nil {
		return fmt.Errorf("Error pulling image, registry: '%s', repository: '%s', tag: '%s', error : %s\n", pullOpts.Registry, pullOpts.Repository, pullOpts.Tag, err)
	}

	return fetchLocalImages(data, client)
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

	foundImage := searchLocalImages(data, imageName)

	if d.Get("keep_updated").(bool) || foundImage == nil {
		if v, ok := d.GetOk("registry"); ok {
			authConfigurations, err := deserializeDockerRegistryConfigurations(v.(string))

			if err != nil {
				return nil, fmt.Errorf("Unable to deserialize registry configuration: %s", err)
			}

			data.DockerRegistry = authConfigurations
		}

		if err := pullImage(&data, client, imageName); err != nil {
			return nil, fmt.Errorf("Unable to pull image %s: %s", imageName, err)
		}
	}

	foundImage = searchLocalImages(data, imageName)
	if foundImage != nil {
		return foundImage, nil
	}

	return nil, fmt.Errorf("Unable to find or pull image %s", imageName)
}
