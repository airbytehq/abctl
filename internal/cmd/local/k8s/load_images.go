package k8s

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/airbytehq/abctl/internal/cmd/local/docker"
	"github.com/airbytehq/abctl/internal/common"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/pterm/pterm"
	nodeslib "sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// loadImages pulls and loads images into the kind cluster.
// It will pull all images in parallel, skip any images that already exist on the nodes,
// save the rest to an image archive (tar file), and load archive onto the nodes.
func loadImages(ctx context.Context, dockerClient docker.Client, nodes []nodeslib.Node, images []string) error {

	// Pull all the images via "docker pull", in parallel.
	var wg sync.WaitGroup
	wg.Add(len(images))
	for _, img := range images {
		pterm.Info.Printfln("Pulling image %s", img)

		go func(img string) {
			defer wg.Done()
			r, err := dockerClient.ImagePull(ctx, img, image.PullOptions{})
			if err != nil {
				pterm.Debug.Printfln("error pulling image %s", err)
				// image pull errors are intentionally dropped because we're in a goroutine,
				// and because we don't want to interrupt other image pulls.
			}
			defer r.Close()
			io.Copy(io.Discard, r)
		}(img)
	}
	wg.Wait()

	// The context could be canceled by now. If so, return early.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Determine which images need to be loaded onto the nodes.
	needed := determineImagesForLoading(ctx, dockerClient, images, nodes)
	if len(needed) == 0 {
		return nil
	}

	// Save all the images to an archive, images.tar
	imagesTarPath, err := saveImageArchive(ctx, dockerClient, needed)
	if err != nil {
		return fmt.Errorf("failed to save image archive: %w", err)
	}
	defer os.RemoveAll(imagesTarPath)

	// Load the image archive into the Kind nodes.
	f, err := os.Open(imagesTarPath)
	if err != nil {
		return fmt.Errorf("failed to open image archive: %w", err)
	}
	defer f.Close()

	for _, n := range nodes {
		pterm.Debug.Printfln("loading image archive into kind node %s", n)
		err := nodeutils.LoadImageArchive(n, f)
		if err != nil {
			pterm.Debug.Printfln("%s", err)
		}
	}
	return nil
}

// getExistingImageDigests returns the set of images that already exist on the nodes.
func getExistingImageDigests(ctx context.Context, nodes []nodeslib.Node) common.Set[string] {
	existingByNode := map[string]int{}

	for _, n := range nodes {

		out, err := exec.CombinedOutputLines(n.CommandContext(ctx, "ctr", "--namespace=k8s.io", "images", "list"))
		if err != nil {
			// ignore the error because discovering these images is just an optimization.
			pterm.Debug.Printfln("error discovering existing images: %s %s", err, out)
			continue
		}
		if len(out) < 1 {
			continue
		}

		// the first line is a header. verify the columns we expect, just in case the format ever changes.
		header := strings.Fields(out[0])
		if len(header) < 1 || header[0] != "REF" {
			pterm.Debug.Printfln("unexpected format from ctr image list. skipping node %s.", n)
			continue
		}

		// skip the first line, which is a header.
		for _, l := range out[1:] {
			fields := strings.Fields(l)
			if len(fields) < 1 {
				continue
			}
			ref := fields[0]
			pterm.Debug.Printfln("found existing image with ref %s", ref)
			existingByNode[ref] += 1
		}
	}

	existing := common.Set[string]{}
	for ref, count := range existingByNode {
		if count == len(nodes) {
			existing.Add(ref)
		}
	}
	return existing
}

// determineImagesForLoading gets the IDs of the desired images (using "docker images"),
// subtracts the images that already exist on the nodes, and returns the resulting list.
func determineImagesForLoading(ctx context.Context, dockerClient docker.Client, images []string, nodes []nodeslib.Node) []string {

	// Get the digests of the images that already exist on the nodes.
	existing := getExistingImageDigests(ctx, nodes)
	if existing.Len() == 0 {
		return images
	}

	// Get the digests of the requested images, so we can compare them to the existing images.
	imgFilter := filters.NewArgs()
	for _, img := range images {
		imgFilter.Add("reference", img)
	}

	imgList, err := dockerClient.ImageList(ctx, image.ListOptions{Filters: imgFilter})
	if err != nil {
		// ignore errors from the image digest list – it's an optimization.
		pterm.Debug.Printfln("error getting image digests: %s", err)
		return images
	}

	// Subtract the images that already exist on the nodes.
	var needed []string
	for _, img := range imgList {
		if !existing.Contains(img.ID) {
			pterm.Debug.Printfln("image does not exist: %s %v", img.ID, img.RepoTags)
			for _, tag := range img.RepoTags {
				needed = append(needed, tag)
			}
		} else {
			pterm.Debug.Printfln("image already exists: %s", img.ID)
		}
	}
	return needed
}

func saveImageArchive(ctx context.Context, dockerClient docker.Client, images []string) (string, error) {

	// Setup the tar path where the images will be saved.
	dir, err := fs.TempDir("", "images-tar-")
	if err != nil {
		return "", err
	}

	imagesTarPath := filepath.Join(dir, "images.tar")
	pterm.Debug.Printfln("saving image archive to %s", imagesTarPath)

	wf, err := os.Create(imagesTarPath)
	if err != nil {
		return "", err
	}
	defer wf.Close()

	r, err := dockerClient.ImageSave(ctx, images)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(wf, r); err != nil {
		return "", err
	}

	return imagesTarPath, nil
}
