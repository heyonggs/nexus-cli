package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/eugenmayer/nexus-cli/utils"
	"html"
	"net/http"
	"os"
)

const AcceptHeader = "application/vnd.docker.distribution.manifest.v2+json"

type Registry struct {
	Host       string `toml:"nexus_host"`
	Username   string `toml:"nexus_username"`
	Password   string `toml:"nexus_password"`
	Repository string `toml:"nexus_repository"`
}

type Repositories struct {
	Images []string `json:"repositories"`
}

type ImageTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type ImageManifest struct {
	SchemaVersion int64       `json:"schemaVersion"`
	MediaType     string      `json:"mediaType"`
	Config        LayerInfo   `json:"config"`
	Layers        []LayerInfo `json:"layers"`
}
type LayerInfo struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

func NewRegistry() (Registry, error) {
	r := Registry{}
	configurationPath := utils.ExpandTildeInPath("~/.nexus-cli")
	if _, err := os.Stat(configurationPath); os.IsNotExist(err) {
		return r, errors.New(fmt.Sprintf("Configuration not found at %s - please run 'nexus-cli configure'\n", configurationPath))
	} else if err != nil {
		return r, err
	}

	if _, err := toml.DecodeFile(configurationPath, &r); err != nil {
		return r, err
	}

	// credits https://github.com/mlabouardy/nexus-cli/pull/12/files
	r.Password = html.UnescapeString(r.Password)
	return r, nil
}

func (r Registry) ListImages() ([]string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/_catalog", r.Host, r.Repository)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", AcceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	var repositories Repositories
	json.NewDecoder(resp.Body).Decode(&repositories)

	return repositories.Images, nil
}

func (r Registry) ListTagsByImage(image string) ([]string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/tags/list", r.Host, r.Repository, image)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", AcceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	var imageTags ImageTags
	json.NewDecoder(resp.Body).Decode(&imageTags)

	return imageTags.Tags, nil
}

func (r Registry) ImageManifest(image string, tag string) (ImageManifest, error) {
	var imageManifest ImageManifest
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return imageManifest, err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", AcceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return imageManifest, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return imageManifest, errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	json.NewDecoder(resp.Body).Decode(&imageManifest)

	return imageManifest, nil

}

func (r Registry) DeleteImageByTag(image string, tag string) error {
	sha, err := r.getImageSHA(image, tag)
	if err != nil {
		return err
	}
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, sha)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", AcceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	fmt.Printf("%s:%s has been successfully deleted\n", image, tag)

	return nil
}

func (r Registry) getImageSHA(image string, tag string) (string, error) {
	client := &http.Client{}

	url := fmt.Sprintf("%s/repository/%s/v2/%s/manifests/%s", r.Host, r.Repository, image, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(r.Username, r.Password)
	req.Header.Add("Accept", AcceptHeader)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("HTTP Code: %d", resp.StatusCode))
	}

	return resp.Header.Get("docker-content-digest"), nil
}
