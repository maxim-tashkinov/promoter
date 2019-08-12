package image

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/docker/distribution/digest"
	//"github.com/docker/distribution/manifest"

	//manifestV1 "github.com/docker/distribution/manifest/schema1"
	//manifestV2 "github.com/docker/distribution/manifest/schema2"

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

	// srcManifest, err := srcHub.Manifest(pr.SrcImage, pr.SrcImageTag)
	if err != nil {
		fmt.Println("Failed to download Source Image manifest. Error: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Manifest version:", srcManifestV2.Versioned.SchemaVersion)
	if srcManifestV2.Versioned.SchemaVersion != 2 {
		fmt.Println("manifest V1 is not supported, exit")
		os.Exit(0)
	}

	srcManifestConfig, err := srcHub.GetManifestConfig(pr.SrcImage, srcManifestV2)

	if err != nil {
		fmt.Println("Failed to get ManifestConfig. Error: " + err.Error())
		os.Exit(1)
	}

	srcLayersV2 := srcManifestV2.Layers

	fmt.Println("Optimising upload for manifest V2...")
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

	fmt.Println("Submitting config of image")
	buffer := bytes.NewBuffer(srcManifestConfig)
	err = destHub.UploadLayer(pr.DestImage, srcManifestV2.Config.Digest, buffer)
	fmt.Println("Submitting Image Manifest")
	err = destHub.PutManifestV2(pr.DestImage, pr.DestImageTag, srcManifestV2)

	if err != nil {
		fmt.Println("Manifest update error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Push Complete")
	os.Exit(0)
}
