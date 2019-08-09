package registry

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/distribution/digest"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
)

func (registry *Registry) Manifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", manifestV1.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	signedManifest := &manifestV1.SignedManifest{}
	err = signedManifest.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}

	return signedManifest, nil
}

func (registry *Registry) ManifestV2(repository, reference string) (*manifestV2.DeserializedManifest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", manifestV2.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	deserialized := &manifestV2.DeserializedManifest{}
	err = deserialized.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	ds, _ := deserialized.MarshalJSON()
	buffer := bytes.NewBuffer(ds)
	fmt.Println("deserialized source manifest:\n", buffer, "\n<<<<\n")
	fmt.Println("deserialized source manifest digest:\n", deserialized.Config.Digest.String(), "\n<<<<\n")

	return deserialized, nil
}

func (registry *Registry) ManifestDigest(repository, reference string) (digest.Digest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	resp, err := registry.Client.Head(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}
	return digest.ParseDigest(resp.Header.Get("Docker-Content-Digest"))
}

func (registry *Registry) DeleteManifest(repository string, digest digest.Digest) error {
	url := registry.url("/v2/%s/manifests/%s", repository, digest)
	registry.Logf("registry.manifest.delete url=%s repository=%s reference=%s", url, repository, digest)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

func (registry *Registry) PutManifest(repository, reference string, signedManifest *manifestV1.SignedManifest) error {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.put url=%s repository=%s reference=%s", url, repository, reference)

	body, err := signedManifest.MarshalJSON()
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(body)
	req, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", manifestV1.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}

func (registry *Registry) HeadManifestV2(repository, reference string) error {
	url := registry.url("/v2/%s/blobs/%s", repository, reference)
	registry.Logf("registry.manifest.HEAD url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.Head(url)
	if err != nil {
		return err
	}
	fmt.Println(req.Body)
	//req.Header.Set("Content-Type", manifestV2.MediaTypeManifest)

	return err
}

func (registry *Registry) PutManifestBlobV2(repository string, body []byte, digest string) error {
	uploadUrl := registry.url("/v2/%s/blobs/uploads/a32f50f6-1eb0-40db-afa1-52cdb0aaddac", repository)

	fmt.Println("Try push Manifest as blob")
	registry.Logf("registry.layer.upload manifest url=%s repository=%s digest=%s", uploadUrl, repository, digest)

	buffer := bytes.NewReader(body)
	fmt.Println("PutManifestBlobV2 1")
	upload, err := http.NewRequest("PUT", uploadUrl, buffer)
	fmt.Println("PutManifestBlobV2 2")
	if err != nil {
		return err
	}
	fmt.Println("PutManifestBlobV2 3")
	upload.Header.Set("digest", digest)
	fmt.Println("PutManifestBlobV2 4")
	upload.Header.Set("Content-Type", "application/octet-stream")
	fmt.Println("PutManifestBlobV2 5")
	res, err := registry.Client.Do(upload)
	fmt.Println("PutManifestBlobV2 6", res.StatusCode)
	if err != nil {
		return err
	}

	return err
}

func (registry *Registry) GetManifestConfig(repository string, manifest *manifestV2.DeserializedManifest) ([]byte, error) {
	digest := manifest.Config.Digest.String()
	url := registry.url("/v2/%s/blobs/%s", repository, digest)
	registry.Logf("registry.GetManifestConfig blob url=%s repository=%s digest=%s", url, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	//if err != nil {
	//	return make([]byte, 1tur), err
	//}

	req.Header.Set("Content-Type", manifestV2.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)

	buffer := bytes.NewBuffer(body)

	fmt.Println("BUFF CONFIG")
	fmt.Println(buffer)
	fmt.Println("<<BUFF CONFIG")

	return body, err
}

func (registry *Registry) PutManifestV2(repository, reference string, signedManifest *manifestV2.DeserializedManifest) error {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.put url=%s repository=%s reference=%s", url, repository, reference)

	body, err := signedManifest.MarshalJSON()
	if err != nil {
		return err
	}

	// check manifest exists - tmp
	err = registry.HeadManifestV2(repository, signedManifest.Config.Digest.String())

	// try put manifest to blob
	//err = registry.PutManifestBlobV2(repository, signedManifest)

	buffer := bytes.NewBuffer(body)
	fmt.Println("URL", url, reference)

	fmt.Println("BUFF")
	fmt.Println(buffer)
	fmt.Println("<<BUFF")

	req, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", manifestV2.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
