package image

import (
	"io/ioutil"
	"log"

	"fmt"
	"os"

	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest"

	//manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"

	//"github.com/docker/libtrust"
	"github.com/dustin/go-humanize"
	"github.com/vbaksa/promoter/connection"
	"github.com/vbaksa/promoter/layer"

	"gopkg.in/cheggaaa/pb.v1"
)

//Promote holds promotion structure used to hold promotion parameters
type Promote struct {
	SrcRegistry  string
	SrcImage     string
	SrcImageTag  string
	SrcUsername  string
	SrcPassword  string
	SrcInsecure  bool
	DestRegistry string
	DestImage    string
	DestImageTag string
	DestUsername string
	DestPassword string
	DestInsecure bool
	Debug        bool
}

//PromoteImage is used to execute specified promotion structure
func (pr *Promote) PromoteImage() {
	if !pr.Debug {
		log.SetOutput(ioutil.Discard)
	}
	fmt.Println("Preparing Image Push")
	srcHub, destHub := connection.InitConnection(pr.SrcRegistry, pr.SrcUsername, pr.SrcPassword, pr.SrcInsecure, pr.DestRegistry, pr.DestUsername, pr.DestPassword, pr.DestInsecure)
	fmt.Println("Source image: " + pr.SrcImage + ":" + pr.SrcImageTag)
	fmt.Println("Destination image: " + pr.DestImage + ":" + pr.DestImageTag)

	srcManifestV2, err := srcHub.ManifestV2(pr.SrcImage, pr.SrcImageTag)
	srcManifest, err := srcHub.Manifest(pr.SrcImage, pr.SrcImageTag)
	if err != nil {
		fmt.Println("Failed to download Source Image manifest. Error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Manifest version:", srcManifestV2.Versioned.SchemaVersion)
	fmt.Println("Manifest media type:", srcManifestV2.Versioned.MediaType)

	srcLayersV2 := srcManifestV2.Layers

	for _, layer_ := range srcLayersV2 {
		fmt.Println(layer_.Digest, layer_.Size, layer_.URLs)
	}
	fmt.Println("Optimising upload for v2...")
	uploadLayerV2 := layer.MissingLayersV2(destHub, pr.DestImage, srcLayersV2)

	if len(uploadLayerV2) > 0 {
		totalDownloadSize := layer.DigestSize(srcHub, pr.SrcImage, uploadLayerV2)
		fmt.Println()
		fmt.Printf("V2 Going to upload around %s of layer data. Expected network bandwidth: %s \n", humanize.Bytes(uint64(totalDownloadSize)), humanize.Bytes(uint64(totalDownloadSize*2)))
		fmt.Println()

		fmt.Println()
		fmt.Println("V2 Uploading layers")
		fmt.Println()

		done := make(chan bool)
		var totalReader = make(chan int64)
		for _, l := range uploadLayerV2 {
			go func(l digest.Digest) {
				fmt.Println("V2 UploadLayerWithProgress", pr.DestImage, l)
				layer.UploadLayerWithProgress(destHub, pr.DestImage, srcHub, pr.SrcImage, l, &totalReader)
				done <- true
			}(l)
		}
		bar := pb.New64(totalDownloadSize * 2).SetUnits(pb.U_BYTES)
		bar.Start()
		go func() {
			for {
				t := <-totalReader
				bar.Add64(t * 2)
			}
		}()

		for i := 0; i < len(uploadLayerV2); i++ {
			<-done
		}
		bar.Finish()

		fmt.Println("Finished uploading layers")
	}

	/*
		fmt.Println("Optimising upload...")
		uploadLayer := layer.MissingLayers(destHub, pr.DestImage, srcLayers)
		if len(uploadLayer) > 0 {
			totalDownloadSize := layer.DigestSize(srcHub, pr.SrcImage, uploadLayer)
			fmt.Println()
			fmt.Printf("Going to upload around %s of layer data. Expected network bandwidth: %s \n", humanize.Bytes(uint64(totalDownloadSize)), humanize.Bytes(uint64(totalDownloadSize*2)))
			fmt.Println()

			fmt.Println()
			fmt.Println("Uploading layers")
			fmt.Println()

			done := make(chan bool)
			var totalReader = make(chan int64)
			for _, l := range uploadLayer {
				go func(l digest.Digest) {
					fmt.Println("No Layer.UploadLayerWithProgress")
					//layer.UploadLayerWithProgress(destHub, pr.DestImage, srcHub, pr.SrcImage, l, &totalReader)
					done <- true
				}(l)
			}
			bar := pb.New64(totalDownloadSize * 2).SetUnits(pb.U_BYTES)
			bar.Start()
			go func() {
				for {
					t := <-totalReader
					bar.Add64(t * 2)
				}
			}()

			for i := 0; i < len(uploadLayer); i++ {
				<-done
			}
			bar.Finish()

			fmt.Println("Finished uploading layers")
		} */
	/*fmt.Println("Generating Signing Key...")
	key, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		fmt.Println("Error occurred while generating Image Key")
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Signing Image Manifest...")*/
	destManifest := srcManifest
	destManifest.Name = pr.DestImage
	destManifest.Tag = pr.DestImageTag

	destManifestV2 := srcManifestV2
	/*
		manifest1 := &manifestV1.Manifest{
			Versioned: manifest.Versioned{
				SchemaVersion: 1,
			},
			Name:         pr.DestImage,
			Tag:          pr.DestImageTag,
			Architecture: destManifest.Architecture,
			FSLayers:     destManifest.FSLayers,
			History:      destManifest.History,
		}
		signedManifest, err := manifestV1.Sign(manifest1, key)
		if err != nil {
			fmt.Println("Error occurred while Signing Image Manifest")
			fmt.Println("Error: " + err.Error())
			os.Exit(1)
		}
	*/
	manifest2 := manifestV2.Manifest{
		Versioned: manifest.Versioned{
			SchemaVersion: 2,
		},
		MediaType: "application/vnd.docker.distribution.manifest.v2+json",
		Layers:    srcLayersV2,
		Config:    destManifestV2.Config,
	}

	deserializedManifest, err := manifestV2.FromStruct(manifest2)
	//fmt.Println(deserializedManifest.UnmarshalJSON())
	//deserializedManifest := manifestV2.DeserializedManifest{
	//	Manifest: *manifest2,
	//}
	//deserializedManifest.UnmarshalJSON(manifest2)

	fmt.Println("Submitting Image Manifest")
	//	err = destHub.PutManifest(pr.DestImage, pr.DestImageTag, signedManifest)
	err = destHub.PutManifestV2(pr.DestImage, pr.DestImageTag, deserializedManifest)
	if err != nil {
		fmt.Println("Manifest update error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Push Complete")
	os.Exit(0)
}
