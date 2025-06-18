package merge

import (
	"fmt"
	"sort"
	"strings"
)

// DockerImages merges two lists of Docker images. Images in b override images in a.
func DockerImages(a, b []string) []string {
	imageMap := make(map[string]string)

	// Add A to map
	for _, img := range a {
		repo, tag := parseDockerImage(img)
		imageMap[repo] = tag
	}

	// Override or add from B
	for _, img := range b {
		repo, tag := parseDockerImage(img)
		imageMap[repo] = tag
	}

	// Reconstruct list
	var result []string
	for repo, tag := range imageMap {
		result = append(result, fmt.Sprintf("%s:%s", repo, tag))
	}

	// Sort for deterministic output
	sort.Strings(result)

	return result
}

// Parse image into repo and tag
func parseDockerImage(image string) (string, string) {
	// Find the last slash to identify where the image name starts
	lastSlash := strings.LastIndex(image, "/")

	// Look for a colon after the last slash (or from the beginning if no slash)
	searchStart := 0
	if lastSlash != -1 {
		searchStart = lastSlash + 1
	}

	// Find the first colon after the last slash (this separates image name from tag)
	colonIndex := strings.Index(image[searchStart:], ":")
	if colonIndex == -1 {
		return image, "latest"
	}

	// Adjust the colon index to be relative to the full string
	colonIndex += searchStart
	return image[:colonIndex], image[colonIndex+1:]
}
